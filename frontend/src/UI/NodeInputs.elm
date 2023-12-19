module UI.NodeInputs exposing
    ( NodeInputOptions
    , nodeCheckboxInput
    , nodeCounterWithReset
    , nodeListInput
    , nodeNumberInput
    , nodeOnOffInput
    , nodeOptionInput
    , nodePasteButton
    , nodeTextInput
    , nodeTimeDateInput
    )

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Color
import Element exposing (..)
import Element.Font as Font
import Element.Input as Input
import List.Extra
import Round
import Svg as S
import Svg.Attributes as Sa
import Time
import Time.Extra
import UI.Button
import UI.Form as Form
import UI.Sanitize as Sanitize
import UI.Style as Style
import Utils.Time exposing (scheduleToLocal, scheduleToUTC)


type alias NodeInputOptions msg =
    { onEditNodePoint : List Point -> msg
    , node : Node
    , now : Time.Posix
    , zone : Time.Zone
    , labelWidth : Int
    }


nodeTextInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeTextInput o key typ lbl placeholder =
    let
        textRaw =
            Point.getText o.node.points typ key
    in
    Input.text
        []
        { onChange =
            \d ->
                o.onEditNodePoint [ Point typ key o.now 0 d 0 ]
        , text =
            if textRaw == "123BLANK123" then
                ""

            else
                let
                    v =
                        Point.getValue o.node.points typ key
                in
                if v /= 0 then
                    ""

                else
                    textRaw
        , placeholder = Just <| Input.placeholder [] <| text placeholder
        , label =
            if lbl == "" then
                Input.labelHidden ""

            else
                Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeTimeDateInput : NodeInputOptions msg -> Int -> Element msg
nodeTimeDateInput o labelWidth =
    let
        zoneOffset =
            Time.Extra.toOffset o.zone o.now

        sModel =
            pointsToSchedule o.node.points

        sLocal =
            checkScheduleToLocal zoneOffset sModel

        sendTime updateSchedule tm =
            let
                tmClean =
                    Sanitize.time tm
            in
            updateSchedule sLocal tmClean
                |> checkScheduleToUTC zoneOffset
                |> scheduleToPoints o.now
                |> o.onEditNodePoint

        updateDate index dUpdate =
            let
                dClean =
                    Sanitize.date dUpdate

                updatedDates =
                    List.Extra.setAt index dClean sLocal.dates

                sUpdate =
                    { sLocal | dates = updatedDates }
            in
            sUpdate
                |> checkScheduleToUTC zoneOffset
                |> scheduleToPoints o.now
                |> o.onEditNodePoint

        deleteDate index =
            let
                updatedDates =
                    List.Extra.removeAt index sLocal.dates

                sUpdate =
                    { sLocal | dates = updatedDates }
            in
            sUpdate
                |> checkScheduleToUTC zoneOffset
                |> scheduleToPoints o.now
                |> o.onEditNodePoint

        weekdaysChecked =
            List.foldl
                (\w ret ->
                    (w /= 0) || ret
                )
                False
                sLocal.weekdays

        weekdayCheckboxInput index label =
            Input.checkbox []
                { onChange =
                    \d ->
                        updateScheduleWkday sLocal index d
                            |> checkScheduleToUTC zoneOffset
                            |> scheduleToPoints o.now
                            |> o.onEditNodePoint
                , checked = List.member index sLocal.weekdays
                , icon = Input.defaultCheckbox
                , label = Input.labelAbove [] <| text label
                }

        dateCount =
            List.length sLocal.dates
    in
    column [ spacing 5 ]
        [ if dateCount <= 0 then
            wrappedRow
                [ spacing 20
                , paddingEach { top = 0, right = 0, bottom = 5, left = 0 }
                ]
                -- here, number matches Go Weekday definitions
                -- https://pkg.go.dev/time#Weekday
                [ el [ width <| px (o.labelWidth - 120) ] none
                , text "Weekdays:"
                , weekdayCheckboxInput 0 "S"
                , weekdayCheckboxInput 1 "M"
                , weekdayCheckboxInput 2 "T"
                , weekdayCheckboxInput 3 "W"
                , weekdayCheckboxInput 4 "T"
                , weekdayCheckboxInput 5 "F"
                , weekdayCheckboxInput 6 "S"
                ]

          else
            none
        , Input.text
            []
            { label = Input.labelLeft [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "Start time:"
            , onChange = sendTime (\sched tm -> { sched | startTime = tm })
            , text = sLocal.startTime
            , placeholder = Nothing
            }
        , Input.text
            []
            { label = Input.labelLeft [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "End time:"
            , onChange = sendTime (\sched tm -> { sched | endTime = tm })
            , text = sLocal.endTime
            , placeholder = Nothing
            }
        , if not weekdaysChecked then
            let
                dateCountS =
                    String.fromInt dateCount
            in
            column []
                [ el [ Element.paddingEach { top = 0, bottom = 0, right = 0, left = labelWidth - 59 } ] <| text "Dates:"
                , column [ spacing 5 ] <|
                    List.indexedMap
                        (\i date ->
                            row [ spacing 10 ]
                                [ Input.text []
                                    { label = Input.labelLeft [ width (px labelWidth) ] <| text ""
                                    , onChange = updateDate i
                                    , text = date
                                    , placeholder = Nothing
                                    }
                                , UI.Button.x <| deleteDate i
                                ]
                        )
                        sLocal.dates
                , el [ Element.paddingEach { top = 6, bottom = 0, right = 0, left = labelWidth - 59 } ] <|
                    Form.button
                        { label = "Add Date"
                        , color = Style.colors.blue
                        , onPress =
                            o.onEditNodePoint
                                [ { typ = Point.typeDate
                                  , key = dateCountS
                                  , text = ""
                                  , time = o.now
                                  , tombstone = 0
                                  , value = 0
                                  }
                                ]
                        }
                ]

          else
            none
        ]


pointsToSchedule : List Point -> Utils.Time.Schedule
pointsToSchedule points =
    let
        start =
            Point.getText points Point.typeStart ""

        end =
            Point.getText points Point.typeEnd ""

        weekdays =
            List.filter
                (\d ->
                    let
                        dString =
                            String.fromInt d

                        p =
                            Point.getValue points Point.typeWeekday dString
                    in
                    p == 1
                )
                [ 0, 1, 2, 3, 4, 5, 6 ]

        datePoints =
            List.filter
                (\p ->
                    p.typ == Point.typeDate && p.tombstone == 0
                )
                points
                |> List.sortWith Point.sort

        dates =
            List.map (\p -> p.text) datePoints
    in
    { startTime = start
    , endTime = end
    , weekdays = weekdays
    , dates = dates
    , dateCount = List.length datePoints
    }


scheduleToPoints : Time.Posix -> Utils.Time.Schedule -> List Point
scheduleToPoints now sched =
    [ Point Point.typeStart "0" now 0 sched.startTime 0
    , Point Point.typeEnd "0" now 0 sched.endTime 0
    ]
        ++ List.map
            (\wday ->
                if List.member wday sched.weekdays then
                    Point Point.typeWeekday (String.fromInt wday) now 1 "" 0

                else
                    Point Point.typeWeekday (String.fromInt wday) now 0 "" 0
            )
            [ 0, 1, 2, 3, 4, 5, 6 ]
        ++ List.indexedMap
            (\i d -> Point Point.typeDate (String.fromInt i) now 0 d 0)
            sched.dates
        ++ (if List.length sched.dates < sched.dateCount then
                -- some dates have been deleted, so send some tombstone points to fill out array length
                List.map
                    (\i ->
                        Point Point.typeDate (String.fromInt i) now 0 "" 1
                    )
                    (List.range (List.length sched.dates) (sched.dateCount - 1))

            else
                []
           )



-- only convert to utc if both times and all dates are valid


checkScheduleToUTC : Int -> Utils.Time.Schedule -> Utils.Time.Schedule
checkScheduleToUTC offset sched =
    if validHM sched.startTime && validHM sched.endTime && validDates sched.dates then
        scheduleToUTC offset sched

    else
        sched


updateScheduleWkday : Utils.Time.Schedule -> Int -> Bool -> Utils.Time.Schedule
updateScheduleWkday sched index checked =
    let
        weekdays =
            if checked then
                if List.member index sched.weekdays then
                    sched.weekdays

                else
                    index :: sched.weekdays

            else
                List.Extra.remove index sched.weekdays
    in
    { sched | weekdays = List.sort weekdays }



-- only convert to local if both times are valid


checkScheduleToLocal : Int -> Utils.Time.Schedule -> Utils.Time.Schedule
checkScheduleToLocal offset sched =
    if validHM sched.startTime && validHM sched.endTime && validDates sched.dates then
        scheduleToLocal offset sched

    else
        sched


validHM : String -> Bool
validHM t =
    case Sanitize.parseHM t of
        Just _ ->
            True

        Nothing ->
            False


validDate : String -> Bool
validDate d =
    case Sanitize.parseDate d of
        Just _ ->
            True

        Nothing ->
            False


validDates : List String -> Bool
validDates dates =
    List.foldl
        (\d ret ->
            if not ret then
                ret

            else if d == "" then
                True

            else
                validDate d
        )
        True
        dates


nodeCheckboxInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> Element msg
nodeCheckboxInput o key typ lbl =
    Input.checkbox
        []
        { onChange =
            \d ->
                let
                    v =
                        if d then
                            1.0

                        else
                            0.0
                in
                o.onEditNodePoint
                    [ Point typ key o.now v "" 0 ]
        , checked =
            Point.getValue o.node.points typ key == 1
        , icon = Input.defaultCheckbox
        , label =
            if lbl /= "" then
                Input.labelLeft [ width (px o.labelWidth) ] <|
                    el [ alignRight ] <|
                        text <|
                            lbl
                                ++ ":"

            else
                Input.labelHidden ""
        }


nodeNumberInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> Element msg
nodeNumberInput o key typ lbl =
    let
        pMaybe =
            Point.get o.node.points typ key

        currentValue =
            case pMaybe of
                Just p ->
                    if p.text /= "" then
                        if p.text == Point.blankMajicValue || p.text == "blank" then
                            ""

                        else if p.text == "-" then
                            "-"

                        else
                            Sanitize.float p.text

                    else
                        String.fromFloat (Round.roundNum 6 p.value)

                Nothing ->
                    ""

        currentValueF =
            case pMaybe of
                Just p ->
                    p.value

                Nothing ->
                    0
    in
    Input.text
        []
        { onChange =
            \d ->
                let
                    dCheck =
                        if d == "" then
                            Point.blankMajicValue

                        else if d == "-" then
                            "-"

                        else
                            case String.toFloat d of
                                Just _ ->
                                    d

                                Nothing ->
                                    currentValue

                    v =
                        if dCheck == Point.blankMajicValue || dCheck == "-" then
                            0

                        else
                            Maybe.withDefault currentValueF <| String.toFloat dCheck
                in
                o.onEditNodePoint
                    [ Point typ key o.now v dCheck 0 ]
        , text = currentValue
        , placeholder = Nothing
        , label =
            if lbl == "" then
                Input.labelHidden ""

            else
                Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeOptionInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> List ( String, String )
    -> Element msg
nodeOptionInput o key typ lbl options =
    Input.radio
        [ spacing 6 ]
        { onChange =
            \sel ->
                o.onEditNodePoint
                    [ Point typ key o.now 0 sel 0 ]
        , label =
            Input.labelLeft [ padding 12, width (px o.labelWidth) ] <|
                el [ alignRight ] <|
                    text <|
                        lbl
                            ++ ":"
        , selected = Just <| Point.getText o.node.points typ key
        , options =
            List.map
                (\opt ->
                    Input.option (Tuple.first opt) (text (Tuple.second opt))
                )
                options
        }


nodeCounterWithReset :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeCounterWithReset o key typ pointResetName lbl =
    let
        currentValue =
            Point.getValue o.node.points typ key

        currentResetValue =
            Point.getValue o.node.points pointResetName key /= 0
    in
    row [ spacing 20 ]
        [ el [ width (px o.labelWidth) ] <|
            el [ alignRight ] <|
                text <|
                    lbl
                        ++ ": "
                        ++ String.fromFloat currentValue
        , Input.checkbox []
            { onChange =
                \v ->
                    let
                        vFloat =
                            if v then
                                1.0

                            else
                                0
                    in
                    o.onEditNodePoint [ Point pointResetName key o.now vFloat "" 0 ]
            , icon = Input.defaultCheckbox
            , checked = currentResetValue
            , label =
                Input.labelLeft [] (text "reset")
            }
        ]


nodeOnOffInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeOnOffInput o key typ pointSetName lbl =
    let
        currentValue =
            Point.getValue o.node.points typ key

        currentSetValue =
            Point.getValue o.node.points pointSetName key

        fill =
            if currentSetValue == 0 then
                Color.rgb 0.5 0.5 0.5

            else
                Color.rgb255 50 100 150

        fillS =
            Color.toCssString fill

        offset =
            if currentSetValue == 0 then
                3

            else
                3 + 24

        newValue =
            if currentSetValue == 0 then
                1

            else
                0
    in
    row [ spacing 10 ]
        [ el [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        , Input.button
            []
            { onPress = Just <| o.onEditNodePoint [ Point pointSetName key o.now newValue "" 0 ]
            , label =
                el [ width (px 100) ] <|
                    html <|
                        S.svg [ Sa.viewBox "0 0 48 24" ]
                            [ S.rect
                                [ Sa.x "0"
                                , Sa.y "0"
                                , Sa.width "48"
                                , Sa.height "24"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill fillS
                                ]
                              <|
                                if currentValue /= currentSetValue then
                                    let
                                        fillFade =
                                            if currentSetValue == 0 then
                                                Color.rgb 0.9 0.9 0.9

                                            else
                                                Color.rgb255 150 200 255

                                        fillFadeS =
                                            Color.toCssString fillFade
                                    in
                                    [ S.animate
                                        [ Sa.attributeName "fill"
                                        , Sa.dur "2s"
                                        , Sa.repeatCount "indefinite"
                                        , Sa.values <|
                                            fillFadeS
                                                ++ ";"
                                                ++ fillS
                                                ++ ";"
                                                ++ fillFadeS
                                        ]
                                        []
                                    ]

                                else
                                    []
                            , S.rect
                                [ Sa.x (String.fromFloat offset)
                                , Sa.y "3"
                                , Sa.width "18"
                                , Sa.height "18"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill (Color.toCssString Color.white)
                                ]
                                []
                            ]
            }
        ]


nodePasteButton :
    NodeInputOptions msg
    -> Element msg
    -> String
    -> String
    -> Element msg
nodePasteButton o label typ value =
    row [ spacing 10, paddingEach { top = 0, bottom = 0, right = 0, left = 75 } ]
        [ UI.Button.clipboard <| o.onEditNodePoint [ Point typ "0" o.now 0 value 0 ]
        , label
        ]


nodeListInput : NodeInputOptions msg -> String -> String -> String -> Element msg
nodeListInput o typ label buttonLabel =
    let
        entries =
            Point.getTextArray o.node.points typ

        entriesArrayCount =
            List.Extra.count
                (\p ->
                    p.typ == typ
                )
                o.node.points

        entriesArrayCountS =
            String.fromInt entriesArrayCount

        entriesToPoints es =
            List.indexedMap
                (\i s ->
                    Point typ (String.fromInt i) o.now 0 s 0
                )
                es
                ++ List.map
                    (\i ->
                        Point typ (String.fromInt i) o.now 0 "" 1
                    )
                    (List.range (List.length es) (entriesArrayCount - 1))

        updateEntry i update =
            List.Extra.setAt i update entries |> entriesToPoints |> o.onEditNodePoint

        deleteEntry i =
            List.Extra.removeAt i entries |> entriesToPoints |> o.onEditNodePoint
    in
    column [ centerX, spacing 5 ] <|
        (el [ Font.bold, centerX, Element.paddingXY 0 6 ] <|
            Element.text label
        )
            :: List.indexedMap
                (\i s ->
                    row [ spacing 10 ]
                        [ Input.text []
                            { label = Input.labelHidden "entry name"
                            , onChange = updateEntry i
                            , text = s
                            , placeholder = Nothing
                            }
                        , UI.Button.x <| deleteEntry i
                        ]
                )
                entries
            ++ [ el [ Element.paddingXY 0 6, centerX ] <|
                    Form.button
                        { label = buttonLabel
                        , color = Style.colors.blue
                        , onPress =
                            o.onEditNodePoint
                                [ { typ = typ
                                  , key = entriesArrayCountS
                                  , text = ""
                                  , time = o.now
                                  , tombstone = 0
                                  , value = 0
                                  }
                                ]
                        }
               ]
