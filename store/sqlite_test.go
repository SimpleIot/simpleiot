package store

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/simpleiot/simpleiot/data"
)

var testFile = "test.sqlite"

func newTestDb(t *testing.T) *DbSqlite {
	_ = exec.Command("sh", "-c", "rm "+testFile+"*").Run()

	db, err := NewSqliteDb(testFile, "")
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}

	return db
}

func TestDbSqlite(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	rootID := db.rootNodeID()

	if rootID == "" {
		t.Fatal("Root ID is blank: ", rootID)
	}

	rns, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	rn := rns[0]

	if rn.ID == "" {
		t.Fatal("Root node ID is blank")
	}

	// modify a point and see if it changes
	err = db.nodePoints(rootID, data.Points{{Type: data.PointTypeDescription, Text: "root"}})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have been root, got: ", rn.Desc())
	}

	// send an old point and verify it does not change
	err = db.nodePoints(rootID, data.Points{{Time: time.Now().Add(-time.Hour),
		Type: data.PointTypeDescription, Text: "root with old time"}})
	if err != nil {
		t.Fatal(err)
	}

	rns, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}
	rn = rns[0]

	if rn.Desc() != "root" {
		t.Fatal("Description should have stayed root, got: ", rn.Desc())
	}

	// verify default admin user got set
	children, err := db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error: ", err)
	}

	if len(children) < 1 {
		t.Fatal("did not return any children")
	}

	if children[0].Parent != rootID {
		t.Fatal("Parent not correct: ", children[0].Parent)
	}

	// test getNodes API
	adminID := children[0].ID

	adminNodes, err := db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	if adminNodes[0].Type != data.NodeTypeUser {
		t.Fatal("getNodes did not return right node type for user")
	}

	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}

	rootNodes, err := db.getNodes(nil, "root", "all", "", false)
	if err != nil {
		t.Fatal("Error getting root nodes", err)
	}

	if len(rootNodes) < 1 {
		t.Fatal("did not return root nodes")
	}

	if rootNodes[0].ID != rootID {
		t.Fatal("root node ID is not correct")
	}

	// test edge points
	err = db.edgePoints(adminID, rootID, data.Points{{Type: data.PointTypeRole, Text: data.PointValueRoleAdmin}})
	if err != nil {
		t.Fatal("Error sending edge points: ", err)
	}

	adminNodes, err = db.getNodes(nil, rootID, adminID, "", false)
	if err != nil {
		t.Fatal("Error getting admin nodes", err)
	}

	p, ok := adminNodes[0].EdgePoints.Find(data.PointTypeRole, "")
	if !ok {
		t.Fatal("point not found")
	}
	if p.Text != data.PointValueRoleAdmin {
		t.Fatal("point does not have right value")
	}

	// try two children
	groupNodeID := uuid.New().String()

	err = db.edgePoints(groupNodeID, rootID, data.Points{
		{Type: data.PointTypeTombstone, Value: 0},
		{Type: data.PointTypeNodeType, Text: data.NodeTypeGroup},
	})
	if err != nil {
		t.Fatal("Error creating group edge", err)
	}

	// verify default admin user got set
	children, err = db.getNodes(nil, rootID, "all", "", false)
	if err != nil {
		t.Fatal("children error: ", err)
	}

	if len(children) < 2 {
		t.Fatal("did not return 2 children")
	}

	// verify getNodes with "all" works
	start := time.Now()
	adminNodes, err = db.getNodes(nil, "all", adminID, "", false)
	fmt.Println("getNodes time: ", time.Since(start))
	if err != nil {
		t.Fatal("Error getting admin nodes with all specified: ", err)
	}

	if adminNodes[0].Parent != rootID {
		t.Fatal("Parent ID is not correct")
	}

	if len(adminNodes) < 1 {
		t.Fatal("did not return admin nodes")
	}
}

func TestDbSqliteKeyZero(t *testing.T) {
	// what to do when we have points with key set to "0" and ""
	// technically these should map to the same point so that
	// we can easily upgrade from scalars to arrays with no
	// data changes
	db := newTestDb(t)
	defer db.Close()

	rootID := db.rootNodeID()

	err := db.nodePoints(rootID, data.Points{{Type: data.PointTypeValue, Value: 1}})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	n := nodes[0]

	err = db.nodePoints(rootID, data.Points{{Type: data.PointTypeValue, Key: "0", Value: 2}})
	if err != nil {
		t.Fatal(err)
	}

	nodes, err = db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	n = nodes[0]

	if len(n.Points) != 1 {
		t.Fatal("Error, point did not get merged")
	}
}

func TestDbSqliteReopen(t *testing.T) {
	db := newTestDb(t)
	rootID := db.rootNodeID()
	db.Close()

	var err error
	db, err = NewSqliteDb(testFile, "")
	if err != nil {
		t.Fatal("Error opening db: ", err)
	}
	defer db.Close()

	if rootID != db.rootNodeID() {
		t.Fatal("Root node ID changed")
	}
}

func TestDbSqliteUserCheck(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	nodes, err := db.userCheck("admin@admin.com", "admin")
	if err != nil {
		t.Fatal("userCheck returned error: ", err)
	}

	if len(nodes) < 1 {
		t.Fatal("userCheck did not return nodes")
	}
}

func TestDbSqliteUp(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	rootID := db.rootNodeID()

	children, err := db.getNodes(nil, rootID, "all", "", false)

	if err != nil {
		t.Fatal("Error getting children")
	}

	if len(children) < 1 {
		t.Fatal("no children")
	}

	childID := children[0].ID

	ups, err := db.up(childID, false)

	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for admin user")
	}

	if ups[0] != rootID {
		t.Fatal("ups, wrong ID: ", ups[0])
	}

	// try to get ups of root node
	ups, err = db.up(rootID, false)

	if err != nil {
		t.Fatal(err)
	}

	if len(ups) < 1 {
		t.Fatal("No ups for root user")
	}

	if ups[0] != "root" {
		t.Fatal("ups, wrong ID for root: ", ups[0])
	}
}

func TestDbSqliteBatchPoints(t *testing.T) {
	db := newTestDb(t)
	defer db.Close()

	rootID := db.rootNodeID()

	now := time.Now()

	pts := data.Points{
		{Time: now, Type: data.PointTypeValue},
		{Time: now.Add(-time.Second), Type: data.PointTypeValue},
		{Time: now.Add(-time.Second * 2), Type: data.PointTypeValue},
	}

	err := db.nodePoints(rootID, pts)
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := db.getNodes(nil, "all", rootID, "", false)
	if err != nil {
		t.Fatal("Error getting root node: ", err)
	}

	n := nodes[0]

	if len(n.Points) != 1 {
		t.Fatal("Error, point did not get merged")
	}

	if !n.Points[0].Time.Equal(now) {
		t.Fatal("Point collapsing did not pick latest")
	}

}
