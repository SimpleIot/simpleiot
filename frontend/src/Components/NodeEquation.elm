module Components.NodeEquation exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            150

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts "" 0

        optionInput =
            NodeInputs.nodeOptionInput opts "" 0

        numberInput =
            NodeInputs.nodeNumberInput opts "" 0

        onOffInput =
            NodeInputs.nodeOnOffInput opts "" 0

        value =
            Point.getValue o.node.points "" 0 Point.typeValue

        variableType =
            Point.getText o.node.points "" 0 Point.typeVariableType

        valueText =
            if variableType == Point.valueNumber then
                String.fromFloat (Round.roundNum 2 value)

            else if value == 0 then
                "off"

            else
                "on"

        valueBackgroundColor =
            if valueText == "on" then
                Style.colors.blue

            else
                Style.colors.none

        valueTextColor =
            if valueText == "on" then
                Style.colors.white

            else
                Style.colors.black
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.divideCircle
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
            , el [ paddingXY 7 0, Background.color valueBackgroundColor, Font.color valueTextColor ] <|
                text <|
                    valueText
                        ++ (if variableType == Point.valueNumber then
                                " " ++ Point.getText o.node.points "" 0 Point.typeUnits

                            else
                                ""
                           )
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    , textInput Point.typeTags "Tags"
                    , textInput Point.typeEquation "Equation"
                    , optionInput Point.typeVariableType
                        "Variable type"
                        [ ( Point.valueOnOff, "On/Off" )
                        , ( Point.valueNumber, "Number" )
                        ]
                    , viewIf (variableType == Point.valueOnOff) <|
                        onOffInput
                            Point.typeValue
                            Point.typeValue
                            "Value"
                    , viewIf (variableType == Point.valueNumber) <|
                        numberInput Point.typeValue "Value"
                    , viewIf (variableType == Point.valueNumber) <|
                        textInput Point.typeUnits "Units"
                    ]

                else
                    []
               )
