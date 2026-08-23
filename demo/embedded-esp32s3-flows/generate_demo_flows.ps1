[CmdletBinding()]
param(
    [string]$ParamsPath,
    [string]$OutDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path

if ([string]::IsNullOrWhiteSpace($ParamsPath)) {
    $localParams = Join-Path $scriptRoot "flow_params.local.json"
    $ParamsPath = if (Test-Path -LiteralPath $localParams) {
        $localParams
    } else {
        Join-Path $scriptRoot "flow_params.example.json"
    }
}

if (-not [System.IO.Path]::IsPathRooted($ParamsPath)) {
    $ParamsPath = Join-Path $scriptRoot $ParamsPath
}

if ([string]::IsNullOrWhiteSpace($OutDir)) {
    $OutDir = Join-Path $scriptRoot "out"
} elseif (-not [System.IO.Path]::IsPathRooted($OutDir)) {
    $OutDir = Join-Path $scriptRoot $OutDir
}

$uuidPattern = "^[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}$"

function Read-Params {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Params file not found: $Path"
    }

    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Require-String {
    param(
        [object]$Value,
        [string]$Label
    )

    $text = [string]$Value
    if ([string]::IsNullOrWhiteSpace($text)) {
        throw "$Label is required."
    }
    return $text.Trim()
}

function Read-OptionalString {
    param([object]$Value)

    if ($null -eq $Value) {
        return ""
    }

    return ([string]$Value).Trim()
}

function Get-ObjectPropertyValue {
    param(
        [object]$Object,
        [string]$Name
    )

    if ($null -eq $Object) {
        return $null
    }

    $prop = $Object.PSObject.Properties[$Name]
    if ($null -eq $prop) {
        return $null
    }

    return $prop.Value
}

function Require-PositiveInt {
    param(
        [object]$Value,
        [string]$Label
    )

    try {
        $number = [int]$Value
    } catch {
        throw "$Label must be an integer."
    }

    if ($number -le 0) {
        throw "$Label must be greater than 0."
    }

    return $number
}

function Require-Visibility {
    param([object]$Value)

    $visibility = Require-String $Value "visibility"
    if ($visibility -notin @("public", "private")) {
        throw "visibility must be 'public' or 'private'."
    }
    return $visibility
}

function Require-FlowId {
    param(
        [object]$Value,
        [string]$Label
    )

    $flowId = Require-String $Value $Label
    if ($flowId -notmatch $uuidPattern) {
        throw "$Label must be a UUID."
    }
    return $flowId.ToLowerInvariant()
}

function Get-Board {
    param(
        [pscustomobject]$Params,
        [string]$BoardKey
    )

    if ($BoardKey -notin @("board1", "board2")) {
        throw "Unsupported board role target '$BoardKey'."
    }

    $board = $Params.$BoardKey
    if ($null -eq $board) {
        throw "Missing board definition: $BoardKey"
    }

    $result = [ordered]@{
        key                = $BoardKey
        label              = Require-String (Get-ObjectPropertyValue -Object $board -Name "label") "$BoardKey.label"
        profile            = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "profile")
        node_id            = Require-PositiveInt (Get-ObjectPropertyValue -Object $board -Name "node_id") "$BoardKey.node_id"
        button12_var       = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "button12_var")
        button13_var       = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "button13_var")
        dht11_temperature_var = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "dht11_temperature_var")
        led_on_var         = Require-String (Get-ObjectPropertyValue -Object $board -Name "led_on_var") "$BoardKey.led_on_var"
        led_red_var        = Require-String (Get-ObjectPropertyValue -Object $board -Name "led_red_var") "$BoardKey.led_red_var"
        led_green_var      = Require-String (Get-ObjectPropertyValue -Object $board -Name "led_green_var") "$BoardKey.led_green_var"
        led_blue_var       = Require-String (Get-ObjectPropertyValue -Object $board -Name "led_blue_var") "$BoardKey.led_blue_var"
        led_brightness_var = Require-String (Get-ObjectPropertyValue -Object $board -Name "led_brightness_var") "$BoardKey.led_brightness_var"
        relay_gpio4_var    = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "relay_gpio4_var")
        relay_gpio5_var    = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "relay_gpio5_var")
        relay_gpio6_var    = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "relay_gpio6_var")
        buzzer_var         = Read-OptionalString (Get-ObjectPropertyValue -Object $board -Name "buzzer_var")
    }

    return [pscustomobject]$result
}

function Require-BoardVar {
    param(
        [pscustomobject]$Board,
        [string]$FieldName
    )

    $value = Read-OptionalString $Board.$FieldName
    if ([string]::IsNullOrWhiteSpace($value)) {
        throw "$($Board.key).$FieldName is required for the selected flow role."
    }
    return $value
}

function Get-RoleBoard {
    param(
        [pscustomobject]$Params,
        [string]$RoleName,
        [hashtable]$BoardMap
    )

    $roleValue = Require-String $Params.roles.$RoleName "roles.$RoleName"
    if (-not $BoardMap.ContainsKey($roleValue)) {
        throw "roles.$RoleName must reference board1 or board2."
    }
    return $BoardMap[$roleValue]
}

function New-UI {
    param(
        [int]$X,
        [int]$Y
    )

    return [ordered]@{
        x = $X
        y = $Y
    }
}

function New-Edge {
    param(
        [string]$From,
        [string]$To,
        [string]$CaseName = ""
    )

    $edge = [ordered]@{
        from = $From
        to   = $To
    }

    if (-not [string]::IsNullOrWhiteSpace($CaseName)) {
        $edge.case = $CaseName
    }

    return $edge
}

function New-CallNode {
    param(
        [string]$Id,
        [string]$Method,
        [hashtable]$ArgsTemplate,
        [int]$X,
        [int]$Y,
        [object[]]$Inputs = @(),
        [Nullable[int]]$Target = $null,
        [int]$TimeoutMs = 5000
    )

    $spec = [ordered]@{
        method        = $Method
        args_template = $ArgsTemplate
        _ui           = (New-UI -X $X -Y $Y)
    }

    if ($null -ne $Target -and $Target.Value -gt 0) {
        $spec.target = $Target.Value
    }

    if ($Inputs.Count -gt 0) {
        $spec.inputs = $Inputs
    }

    return [ordered]@{
        id               = $Id
        kind             = "call"
        allow_fail       = $false
        retry            = 0
        retry_backoff_ms = 0
        timeout_ms       = $TimeoutMs
        spec             = $spec
    }
}

function New-BranchNode {
    param(
        [string]$Id,
        [string]$SourceNodeId,
        [string]$SourcePath,
        [string]$MatchValue,
        [string]$MatchCaseName,
        [string]$DefaultCase,
        [int]$X,
        [int]$Y
    )

    return [ordered]@{
        id               = $Id
        kind             = "branch"
        allow_fail       = $false
        retry            = 0
        retry_backoff_ms = 0
        timeout_ms       = 0
        spec             = [ordered]@{
            cases = @(
                [ordered]@{
                    name  = $MatchCaseName
                    match = [ordered]@{
                        source = [ordered]@{
                            kind    = "node_result"
                            node_id = $SourceNodeId
                            path    = $SourcePath
                        }
                        op    = "eq"
                        value = $MatchValue
                    }
                },
                [ordered]@{
                    name  = $DefaultCase
                    match = [ordered]@{
                        source = [ordered]@{
                            kind    = "node_result"
                            node_id = $SourceNodeId
                            path    = $SourcePath
                        }
                        op = "exists"
                    }
                }
            )
            default_case = $DefaultCase
            _ui          = (New-UI -X $X -Y $Y)
        }
    }
}

function New-NodeResultCasesBranchNode {
    param(
        [string]$Id,
        [string]$SourceNodeId,
        [string]$SourcePath,
        [object[]]$Cases,
        [string]$DefaultCase,
        [int]$X,
        [int]$Y
    )

    if ($Cases.Count -eq 0) {
        throw "Branch node '$Id' requires at least one case."
    }

    $caseSpecs = [System.Collections.Generic.List[object]]::new()
    foreach ($branchCase in $Cases) {
        $caseName = Require-String $branchCase.name "branch.case.name"
        $matchValue = Require-String $branchCase.value "branch.case.value"
        $caseSpecs.Add([ordered]@{
                name  = $caseName
                match = [ordered]@{
                    source = [ordered]@{
                        kind    = "node_result"
                        node_id = $SourceNodeId
                        path    = $SourcePath
                    }
                    op    = "eq"
                    value = $matchValue
                }
            })
    }
    $caseSpecs.Add([ordered]@{
            name  = $DefaultCase
            match = [ordered]@{
                source = [ordered]@{
                    kind    = "node_result"
                    node_id = $SourceNodeId
                    path    = $SourcePath
                }
                op = "exists"
            }
        })

    return [ordered]@{
        id               = $Id
        kind             = "branch"
        allow_fail       = $false
        retry            = 0
        retry_backoff_ms = 0
        timeout_ms       = 0
        spec             = [ordered]@{
            cases        = $caseSpecs.ToArray()
            default_case = $DefaultCase
            _ui          = (New-UI -X $X -Y $Y)
        }
    }
}

function New-TriggerBranchNode {
    param(
        [string]$Id,
        [string]$TriggerPath,
        [string]$MatchValue,
        [string]$MatchCaseName,
        [string]$DefaultCase,
        [int]$X,
        [int]$Y
    )

    return [ordered]@{
        id               = $Id
        kind             = "branch"
        allow_fail       = $false
        retry            = 0
        retry_backoff_ms = 0
        timeout_ms       = 0
        spec             = [ordered]@{
            cases = @(
                [ordered]@{
                    name  = $MatchCaseName
                    match = [ordered]@{
                        source = [ordered]@{
                            kind = "trigger"
                            path = $TriggerPath
                        }
                        op    = "eq"
                        value = $MatchValue
                    }
                },
                [ordered]@{
                    name  = $DefaultCase
                    match = [ordered]@{
                        source = [ordered]@{
                            kind = "trigger"
                            path = $TriggerPath
                        }
                        op = "exists"
                    }
                }
            )
            default_case = $DefaultCase
            _ui          = (New-UI -X $X -Y $Y)
        }
    }
}

function New-VarStoreGetNode {
    param(
        [string]$Id,
        [int]$Owner,
        [string]$Name,
        [int]$X,
        [int]$Y
    )

    return New-CallNode -Id $Id -Method "varstore::get" -ArgsTemplate ([ordered]@{
            owner = $Owner
            name  = $Name
        }) -X $X -Y $Y
}

function New-VarStoreSetNode {
    param(
        [string]$Id,
        [int]$Owner,
        [string]$Name,
        [string]$Value,
        [string]$Type,
        [string]$Visibility,
        [int]$X,
        [int]$Y
    )

    return New-CallNode -Id $Id -Method "varstore::set" -ArgsTemplate ([ordered]@{
            owner      = $Owner
            name       = $Name
            value      = $Value
            type       = $Type
            visibility = $Visibility
        }) -X $X -Y $Y
}

function Add-ActionChain {
    param(
        [System.Collections.Generic.List[object]]$Nodes,
        [System.Collections.Generic.List[object]]$Edges,
        [string]$EntryNodeId,
        [string]$EntryCase,
        [int]$StartX,
        [int]$StartY,
        [object[]]$Actions,
        [string]$Visibility
    )

    if ($Actions.Count -eq 0) {
        return
    }

    $previousId = $EntryNodeId
    $index = 0
    foreach ($action in $Actions) {
        $nodeId = Require-String $action.id "action.id"
        $board = $action.board
        $varField = Require-String $action.varField "action.varField"
        $name = Require-String $board.$varField "$($board.key).$varField"
        $value = Require-String $action.value "action.value"
        $type = Require-String $action.type "action.type"
        $node = New-VarStoreSetNode -Id $nodeId `
            -Owner (Require-PositiveInt $board.node_id "$($board.key).node_id") `
            -Name $name `
            -Value $value `
            -Type $type `
            -Visibility $Visibility `
            -X ($StartX + ($index * 220)) `
            -Y $StartY
        $Nodes.Add($node)

        if ($index -eq 0) {
            $Edges.Add((New-Edge -From $EntryNodeId -To $nodeId -CaseName $EntryCase))
        } else {
            $Edges.Add((New-Edge -From $previousId -To $nodeId))
        }

        $previousId = $nodeId
        $index++
    }
}

function Get-TemperatureAlarmSettings {
    param([pscustomobject]$Params)

    $settings = Get-ObjectPropertyValue -Object $Params -Name "temperature_alarm"
    if ($null -eq $settings) {
        throw "temperature_alarm is required."
    }

    $thresholdC = Require-PositiveInt (Get-ObjectPropertyValue -Object $settings -Name "threshold_c") "temperature_alarm.threshold_c"
    $maxC = Require-PositiveInt (Get-ObjectPropertyValue -Object $settings -Name "max_c") "temperature_alarm.max_c"
    if ($thresholdC -gt $maxC) {
        throw "temperature_alarm.threshold_c must be less than or equal to temperature_alarm.max_c."
    }

    return [pscustomobject]@{
        threshold_c = $thresholdC
        max_c       = $maxC
    }
}

function New-FlowPayload {
    param(
        [string]$FlowId,
        [string]$Name,
        [hashtable]$Trigger,
        [object[]]$Nodes,
        [object[]]$Edges
    )

    return [ordered]@{
        flow_id         = $FlowId
        name            = $Name
        max_active_runs = 1
        trigger         = $Trigger
        graph           = [ordered]@{
            nodes = $Nodes
            edges = $Edges
        }
    }
}

function New-ButtonSceneFlow {
    param(
        [pscustomobject]$Params,
        [hashtable]$BoardMap,
        [string]$Visibility
    )

    $buttonBoard = Get-RoleBoard -Params $Params -RoleName "button_board" -BoardMap $BoardMap
    $sceneBoard = Get-RoleBoard -Params $Params -RoleName "scene_board" -BoardMap $BoardMap
    $flowId = Require-FlowId $Params.flow_ids.button_scene "flow_ids.button_scene"
    $button12Var = Require-BoardVar -Board $buttonBoard -FieldName "button12_var"
    $buttonRelay4Var = Require-BoardVar -Board $buttonBoard -FieldName "relay_gpio4_var"
    $buttonBuzzerVar = Require-BoardVar -Board $buttonBoard -FieldName "buzzer_var"

    $nodes = [System.Collections.Generic.List[object]]::new()
    $edges = [System.Collections.Generic.List[object]]::new()

    $nodes.Add((New-VarStoreGetNode -Id "read_button" -Owner $buttonBoard.node_id -Name $button12Var -X 40 -Y 0))
    $nodes.Add((New-BranchNode -Id "route_button" -SourceNodeId "read_button" -SourcePath "/value" -MatchValue "true" -MatchCaseName "pressed" -DefaultCase "released" -X 300 -Y 0))
    $edges.Add((New-Edge -From "read_button" -To "route_button"))

    $pressedActions = @(
        [ordered]@{ id = "pressed_expansion_relay4"; board = $buttonBoard; varField = "relay_gpio4_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "pressed_expansion_buzzer"; board = $buttonBoard; varField = "buzzer_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "pressed_expansion_led_on"; board = $buttonBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "pressed_expansion_led_red"; board = $buttonBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "pressed_expansion_led_green"; board = $buttonBoard; varField = "led_green_var"; value = "255"; type = "u8" },
        [ordered]@{ id = "pressed_expansion_led_blue"; board = $buttonBoard; varField = "led_blue_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "pressed_expansion_led_brightness"; board = $buttonBoard; varField = "led_brightness_var"; value = "32"; type = "u8" },
        [ordered]@{ id = "pressed_core_led_on"; board = $sceneBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "pressed_core_led_red"; board = $sceneBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "pressed_core_led_green"; board = $sceneBoard; varField = "led_green_var"; value = "255"; type = "u8" },
        [ordered]@{ id = "pressed_core_led_blue"; board = $sceneBoard; varField = "led_blue_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "pressed_core_led_brightness"; board = $sceneBoard; varField = "led_brightness_var"; value = "32"; type = "u8" }
    )

    $releasedActions = @(
        [ordered]@{ id = "released_expansion_relay4"; board = $buttonBoard; varField = "relay_gpio4_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "released_expansion_buzzer"; board = $buttonBoard; varField = "buzzer_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "released_expansion_led_on"; board = $buttonBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "released_expansion_led_red"; board = $buttonBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "released_expansion_led_green"; board = $buttonBoard; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "released_expansion_led_blue"; board = $buttonBoard; varField = "led_blue_var"; value = "64"; type = "u8" },
        [ordered]@{ id = "released_expansion_led_brightness"; board = $buttonBoard; varField = "led_brightness_var"; value = "18"; type = "u8" },
        [ordered]@{ id = "released_core_led_on"; board = $sceneBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "released_core_led_red"; board = $sceneBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "released_core_led_green"; board = $sceneBoard; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "released_core_led_blue"; board = $sceneBoard; varField = "led_blue_var"; value = "64"; type = "u8" },
        [ordered]@{ id = "released_core_led_brightness"; board = $sceneBoard; varField = "led_brightness_var"; value = "18"; type = "u8" }
    )

    Add-ActionChain -Nodes $nodes -Edges $edges -EntryNodeId "route_button" -EntryCase "pressed" -StartX 560 -StartY -140 -Actions $pressedActions -Visibility $Visibility
    Add-ActionChain -Nodes $nodes -Edges $edges -EntryNodeId "route_button" -EntryCase "released" -StartX 560 -StartY 140 -Actions $releasedActions -Visibility $Visibility

    return New-FlowPayload -FlowId $flowId -Name "ESP32 Button12 Cross-Board Scene" -Trigger ([ordered]@{
            type            = "var_changed"
            var_owner       = $buttonBoard.node_id
            var_name        = $button12Var
            dedup_window_ms = 300
        }) -Nodes $nodes.ToArray() -Edges $edges.ToArray()
}

function New-TemperatureAlarmFlow {
    param(
        [pscustomobject]$Params,
        [hashtable]$BoardMap,
        [string]$Visibility
    )

    $sensorBoard = Get-RoleBoard -Params $Params -RoleName "sensor_board" -BoardMap $BoardMap
    $sceneBoard = Get-RoleBoard -Params $Params -RoleName "scene_board" -BoardMap $BoardMap
    $flowId = Require-FlowId $Params.flow_ids.temperature_alarm "flow_ids.temperature_alarm"
    $temperatureAlarm = Get-TemperatureAlarmSettings -Params $Params
    $sensorTemperatureVar = Require-BoardVar -Board $sensorBoard -FieldName "dht11_temperature_var"
    $sensorRelay5Var = Require-BoardVar -Board $sensorBoard -FieldName "relay_gpio5_var"
    $sensorBuzzerVar = Require-BoardVar -Board $sensorBoard -FieldName "buzzer_var"

    $nodes = [System.Collections.Generic.List[object]]::new()
    $edges = [System.Collections.Generic.List[object]]::new()

    $nodes.Add((New-TriggerBranchNode -Id "route_trigger_op" -TriggerPath "/op" -MatchValue "deleted" -MatchCaseName "deleted" -DefaultCase "changed" -X 40 -Y 0))
    $nodes.Add((New-VarStoreGetNode -Id "read_temperature" -Owner $sensorBoard.node_id -Name $sensorTemperatureVar -X 300 -Y -120))

    $temperatureCases = [System.Collections.Generic.List[object]]::new()
    $temperatureCaseNames = [System.Collections.Generic.List[string]]::new()
    for ($temperatureC = $temperatureAlarm.threshold_c; $temperatureC -le $temperatureAlarm.max_c; $temperatureC++) {
        $caseName = "temp_$temperatureC"
        $temperatureCaseNames.Add($caseName)
        $temperatureCases.Add([ordered]@{
                name  = $caseName
                value = [string]$temperatureC
            })
    }
    $nodes.Add((New-NodeResultCasesBranchNode -Id "route_temperature" -SourceNodeId "read_temperature" -SourcePath "/value" -Cases $temperatureCases.ToArray() -DefaultCase "normal" -X 560 -Y -120))

    $edges.Add((New-Edge -From "route_trigger_op" -To "read_temperature" -CaseName "changed"))

    $alertActions = @(
        [ordered]@{ id = "alert_expansion_led_on"; board = $sensorBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "alert_expansion_led_red"; board = $sensorBoard; varField = "led_red_var"; value = "255"; type = "u8" },
        [ordered]@{ id = "alert_expansion_led_green"; board = $sensorBoard; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "alert_expansion_led_blue"; board = $sensorBoard; varField = "led_blue_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "alert_expansion_led_brightness"; board = $sensorBoard; varField = "led_brightness_var"; value = "48"; type = "u8" },
        [ordered]@{ id = "alert_expansion_relay5"; board = $sensorBoard; varField = "relay_gpio5_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "alert_expansion_buzzer"; board = $sensorBoard; varField = "buzzer_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "alert_core_led_on"; board = $sceneBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "alert_core_led_red"; board = $sceneBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "alert_core_led_green"; board = $sceneBoard; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "alert_core_led_blue"; board = $sceneBoard; varField = "led_blue_var"; value = "255"; type = "u8" },
        [ordered]@{ id = "alert_core_led_brightness"; board = $sceneBoard; varField = "led_brightness_var"; value = "48"; type = "u8" }
    )

    $normalActions = @(
        [ordered]@{ id = "normal_expansion_relay5"; board = $sensorBoard; varField = "relay_gpio5_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "normal_expansion_buzzer"; board = $sensorBoard; varField = "buzzer_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "normal_expansion_led_on"; board = $sensorBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "normal_expansion_led_red"; board = $sensorBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "normal_expansion_led_green"; board = $sensorBoard; varField = "led_green_var"; value = "180"; type = "u8" },
        [ordered]@{ id = "normal_expansion_led_blue"; board = $sensorBoard; varField = "led_blue_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "normal_expansion_led_brightness"; board = $sensorBoard; varField = "led_brightness_var"; value = "24"; type = "u8" },
        [ordered]@{ id = "normal_core_led_on"; board = $sceneBoard; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "normal_core_led_red"; board = $sceneBoard; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "normal_core_led_green"; board = $sceneBoard; varField = "led_green_var"; value = "180"; type = "u8" },
        [ordered]@{ id = "normal_core_led_blue"; board = $sceneBoard; varField = "led_blue_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "normal_core_led_brightness"; board = $sceneBoard; varField = "led_brightness_var"; value = "24"; type = "u8" }
    )

    Add-ActionChain -Nodes $nodes -Edges $edges -EntryNodeId "route_trigger_op" -EntryCase "deleted" -StartX 300 -StartY 180 -Actions $alertActions -Visibility $Visibility
    $edges.Add((New-Edge -From "read_temperature" -To "route_temperature"))
    Add-ActionChain -Nodes $nodes -Edges $edges -EntryNodeId "route_temperature" -EntryCase "normal" -StartX 820 -StartY -260 -Actions $normalActions -Visibility $Visibility
    foreach ($caseName in $temperatureCaseNames) {
        $edges.Add((New-Edge -From "route_temperature" -To "alert_expansion_led_on" -CaseName $caseName))
    }

    return New-FlowPayload -FlowId $flowId -Name "ESP32 DHT11 Temperature Alarm" -Trigger ([ordered]@{
            type            = "var_changed"
            var_owner       = $sensorBoard.node_id
            var_name        = $sensorTemperatureVar
            dedup_window_ms = 1000
        }) -Nodes $nodes.ToArray() -Edges $edges.ToArray()
}

function New-ResetFlow {
    param(
        [pscustomobject]$Params,
        [hashtable]$BoardMap,
        [string]$Visibility
    )

    $resetBoard = Get-RoleBoard -Params $Params -RoleName "reset_board" -BoardMap $BoardMap
    $board1 = $BoardMap["board1"]
    $board2 = $BoardMap["board2"]
    $flowId = Require-FlowId $Params.flow_ids.reset_scene "flow_ids.reset_scene"
    $resetButton13Var = Require-BoardVar -Board $resetBoard -FieldName "button13_var"
    $board1Relay4Var = Require-BoardVar -Board $board1 -FieldName "relay_gpio4_var"
    $board1Relay5Var = Require-BoardVar -Board $board1 -FieldName "relay_gpio5_var"
    $board1Relay6Var = Require-BoardVar -Board $board1 -FieldName "relay_gpio6_var"
    $board1BuzzerVar = Require-BoardVar -Board $board1 -FieldName "buzzer_var"

    $nodes = [System.Collections.Generic.List[object]]::new()
    $edges = [System.Collections.Generic.List[object]]::new()

    $nodes.Add((New-VarStoreGetNode -Id "read_reset_button" -Owner $resetBoard.node_id -Name $resetButton13Var -X 40 -Y 0))
    $nodes.Add((New-BranchNode -Id "route_reset" -SourceNodeId "read_reset_button" -SourcePath "/value" -MatchValue "true" -MatchCaseName "reset" -DefaultCase "ignore" -X 300 -Y 0))
    $edges.Add((New-Edge -From "read_reset_button" -To "route_reset"))

    $resetActions = @(
        [ordered]@{ id = "reset_expansion_relay4"; board = $board1; varField = "relay_gpio4_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "reset_expansion_relay5"; board = $board1; varField = "relay_gpio5_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "reset_expansion_relay6"; board = $board1; varField = "relay_gpio6_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "reset_expansion_buzzer"; board = $board1; varField = "buzzer_var"; value = "false"; type = "bool" },
        [ordered]@{ id = "reset_expansion_led_on"; board = $board1; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "reset_expansion_led_red"; board = $board1; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "reset_expansion_led_green"; board = $board1; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "reset_expansion_led_blue"; board = $board1; varField = "led_blue_var"; value = "48"; type = "u8" },
        [ordered]@{ id = "reset_expansion_led_brightness"; board = $board1; varField = "led_brightness_var"; value = "18"; type = "u8" },
        [ordered]@{ id = "reset_core_led_on"; board = $board2; varField = "led_on_var"; value = "true"; type = "bool" },
        [ordered]@{ id = "reset_core_led_red"; board = $board2; varField = "led_red_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "reset_core_led_green"; board = $board2; varField = "led_green_var"; value = "0"; type = "u8" },
        [ordered]@{ id = "reset_core_led_blue"; board = $board2; varField = "led_blue_var"; value = "48"; type = "u8" },
        [ordered]@{ id = "reset_core_led_brightness"; board = $board2; varField = "led_brightness_var"; value = "18"; type = "u8" }
    )

    Add-ActionChain -Nodes $nodes -Edges $edges -EntryNodeId "route_reset" -EntryCase "reset" -StartX 560 -StartY 0 -Actions $resetActions -Visibility $Visibility

    return New-FlowPayload -FlowId $flowId -Name "ESP32 Demo Reset" -Trigger ([ordered]@{
            type            = "var_changed"
            var_owner       = $resetBoard.node_id
            var_name        = $resetButton13Var
            dedup_window_ms = 300
        }) -Nodes $nodes.ToArray() -Edges $edges.ToArray()
}

$paramsObject = Read-Params -Path $ParamsPath
$visibility = Require-Visibility $paramsObject.visibility
$executorNode = Require-PositiveInt $paramsObject.executor_node "executor_node"

$boardMap = @{
    board1 = (Get-Board -Params $paramsObject -BoardKey "board1")
    board2 = (Get-Board -Params $paramsObject -BoardKey "board2")
}

foreach ($roleName in @("button_board", "sensor_board", "reset_board", "scene_board")) {
    [void](Get-RoleBoard -Params $paramsObject -RoleName $roleName -BoardMap $boardMap)
}

$flows = @(
    @{
        FileName = "esp32s3_button12_cross_board_scene.json"
        Payload  = (New-ButtonSceneFlow -Params $paramsObject -BoardMap $boardMap -Visibility $visibility)
    },
    @{
        FileName = "esp32s3_dht11_temperature_alarm.json"
        Payload  = (New-TemperatureAlarmFlow -Params $paramsObject -BoardMap $boardMap -Visibility $visibility)
    },
    @{
        FileName = "esp32s3_demo_reset.json"
        Payload  = (New-ResetFlow -Params $paramsObject -BoardMap $boardMap -Visibility $visibility)
    }
)

[void](New-Item -ItemType Directory -Force -Path $OutDir)

foreach ($flow in $flows) {
    $json = $flow.Payload | ConvertTo-Json -Depth 32
    $outPath = Join-Path $OutDir $flow.FileName
    Set-Content -LiteralPath $outPath -Value ($json + [Environment]::NewLine) -Encoding utf8
}

Write-Host "Generated $($flows.Count) flow payloads into $OutDir"
Write-Host "Recommended deploy executor_node: $executorNode"
