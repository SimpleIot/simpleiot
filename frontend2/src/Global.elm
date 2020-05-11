module Global exposing
    ( Flags
    , Model(..)
    , Msg(..)
    , init
    , subscriptions
    , update
    )

import Device as D
import Generated.Routes exposing (Route, routes)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required, resolve)
import Json.Encode as Encode
import Org as O
import Time
import Url.Builder as Url
import User as U


type alias Flags =
    ()


type Model
    = SignedOut (Maybe Http.Error)
    | SignedIn Session


type alias Session =
    { cred : Cred
    , authToken : String
    , privilege : Privilege
    , data : Data
    , error : Maybe Http.Error
    }


emptyData : Data
emptyData =
    { orgs = []
    , users = []
    , devices = []
    }


type alias Data =
    { orgs : List O.Org
    , devices : List D.Device
    , users : List U.User
    }


type alias Cred =
    { email : String
    , password : String
    }


type Msg
    = DevicesResponse (Result Http.Error (List D.Device))
    | OrgsResponse (Result Http.Error (List O.Org))
    | SignIn Cred
    | AuthResponse Cred (Result Http.Error Auth)
    | DataResponse (Result Http.Error Data)
    | RequestOrgs
    | RequestDevices
    | SignOut
    | Tick Time.Posix
    | UpdateDeviceConfig String D.Config
    | ConfigPosted String (Result Http.Error Response)


type alias Commands msg =
    { navigate : Route -> Cmd msg
    }


init : Commands msg -> Flags -> ( Model, Cmd Msg, Cmd msg )
init _ _ =
    ( SignedOut Nothing
    , Cmd.none
    , Cmd.none
    )


login : Cred -> Cmd Msg
login cred =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" cred.email
                , Http.stringPart "password" cred.password
                ]
        , url = Url.absolute [ "v1", "auth" ] []
        , expect = Http.expectJson (AuthResponse cred) decodeAuth
        }


getData : String -> Cmd Msg
getData token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "data" ] []
        , expect = Http.expectJson DataResponse decodeData
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


decodeData : Decode.Decoder Data
decodeData =
    Decode.succeed Data
        |> required "orgs" O.decodeList
        |> required "devices" D.decodeList
        |> required "users" U.decodeList


type alias Auth =
    { token : String
    , privilege : Privilege
    }


type Privilege
    = User
    | Admin
    | Root


decodeAuth : Decode.Decoder Auth
decodeAuth =
    Decode.succeed validate
        |> required "token" Decode.string
        |> required "privilege" Decode.string
        |> resolve


validate : String -> String -> Decode.Decoder Auth
validate token privilege =
    case privilege of
        "user" ->
            Decode.succeed <| Auth token User

        "admin" ->
            Decode.succeed <| Auth token Admin

        "root" ->
            Decode.succeed <| Auth token Root

        _ ->
            Decode.fail "sign in failed"


update : Commands msg -> Msg -> Model -> ( Model, Cmd Msg, Cmd msg )
update commands msg model =
    case model of
        SignedOut _ ->
            case msg of
                SignIn cred ->
                    ( SignedOut Nothing
                    , login cred
                    , Cmd.none
                    )

                AuthResponse cred (Ok { token, privilege }) ->
                    ( SignedIn
                        { authToken = token
                        , cred = cred
                        , privilege = privilege
                        , data = emptyData
                        , error = Nothing
                        }
                    , getData token
                    , commands.navigate routes.top
                    )

                AuthResponse _ (Err error) ->
                    ( SignedOut (Just error), Cmd.none, Cmd.none )

                _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

        SignedIn sess ->
            let
                data =
                    sess.data
            in
            case msg of
                SignOut ->
                    ( SignedOut Nothing
                    , Cmd.none
                    , commands.navigate routes.top
                    )

                AuthResponse _ (Err err) ->
                    ( SignedOut (Just err)
                    , Cmd.none
                    , commands.navigate routes.signIn
                    )

                DataResponse (Ok newData) ->
                    ( SignedIn { sess | data = newData }
                    , Cmd.none
                    , Cmd.none
                    )

                DevicesResponse (Ok devices) ->
                    ( SignedIn { sess | data = { data | devices = devices } }
                    , Cmd.none
                    , Cmd.none
                    )

                OrgsResponse (Ok orgs) ->
                    ( SignedIn { sess | data = { data | orgs = orgs } }
                    , Cmd.none
                    , Cmd.none
                    )

                RequestOrgs ->
                    ( model
                    , getOrgs sess.authToken
                    , Cmd.none
                    )

                Tick _ ->
                    ( model
                    , getDevices sess.authToken
                    , Cmd.none
                    )

                UpdateDeviceConfig id config ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )

                _ ->
                    ( model
                    , Cmd.none
                    , Cmd.none
                    )


getDevices : String -> Cmd Msg
getDevices token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = urlDevices
        , expect = Http.expectJson DevicesResponse D.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


deviceConfigEncoder : D.Config -> Encode.Value
deviceConfigEncoder deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]


responseDecoder : Decode.Decoder Response
responseDecoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""


postConfig : String -> String -> D.Config -> Cmd Msg
postConfig token id config =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "devices", id, "config" ] []
        , expect = Http.expectJson (ConfigPosted id) responseDecoder
        , body = config |> deviceConfigEncoder |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


urlDevices : String
urlDevices =
    Url.absolute [ "v1", "devices" ] []


getOrgs : String -> Cmd Msg
getOrgs token =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ token ]
        , url = Url.absolute [ "v1", "orgs" ] []
        , expect = Http.expectJson OrgsResponse O.decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 10000 Tick
        ]
