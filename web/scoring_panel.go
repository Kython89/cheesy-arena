// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for scoring interface.

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/websocket"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"net/http"
	"strings"
)

type ScoringPosition struct {
	Title            string
	Alliance         string
	NearSide         bool
	ScoresAuto       bool
	ScoresEndgame    bool
	ScoresBarge      bool
	ScoresProcessor  bool
	LeftmostReefPole int
}

var positionParameters = map[string]ScoringPosition{
	"red_near": {
		Title:            "Red Near",
		Alliance:         "red",
		NearSide:         true,
		ScoresAuto:       true,
		ScoresEndgame:    true,
		ScoresBarge:      true,
		ScoresProcessor:  false,
		LeftmostReefPole: 6,
	},
	"red_far": {
		Title:            "Red Far",
		Alliance:         "red",
		NearSide:         false,
		ScoresAuto:       false,
		ScoresEndgame:    false,
		ScoresBarge:      false,
		ScoresProcessor:  true,
		LeftmostReefPole: 0,
	},
	"blue_near": {
		Title:            "Blue Near",
		Alliance:         "blue",
		NearSide:         true,
		ScoresAuto:       false,
		ScoresEndgame:    false,
		ScoresBarge:      false,
		ScoresProcessor:  true,
		LeftmostReefPole: 0,
	},
	"blue_far": {
		Title:            "Blue Far",
		Alliance:         "blue",
		NearSide:         false,
		ScoresAuto:       true,
		ScoresEndgame:    true,
		ScoresBarge:      true,
		ScoresProcessor:  false,
		LeftmostReefPole: 6,
	},
}

// Renders the scoring interface which enables input of scores in real-time.
func (web *Web) scoringPanelHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	position := r.PathValue("position")
	parameters, ok := positionParameters[position]
	if !ok {
		handleWebErr(w, fmt.Errorf("Invalid position '%s'.", position))
		return
	}

	template, err := web.parseFiles("templates/scoring_panel.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		PlcIsEnabled bool
		PositionName string
		Position     ScoringPosition
	}{web.arena.EventSettings, web.arena.Plc.IsEnabled(), position, parameters}
	err = template.ExecuteTemplate(w, "base_no_navbar", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the scoring interface client to send control commands and receive status updates.
func (web *Web) scoringPanelWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	position := r.PathValue("position")
	if position != "red_near" && position != "red_far" && position != "blue_near" && position != "blue_far" {
		handleWebErr(w, fmt.Errorf("Invalid position '%s'.", position))
		return
	}
	alliance := strings.Split(position, "_")[0]

	var realtimeScore **field.RealtimeScore
	if alliance == "red" {
		realtimeScore = &web.arena.RedRealtimeScore
	} else {
		realtimeScore = &web.arena.BlueRealtimeScore
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer ws.Close()
	web.arena.ScoringPanelRegistry.RegisterPanel(position, ws)
	web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringPanelRegistry.UnregisterPanel(position, ws)

	// Instruct panel to clear any local state in case this is a reconnect
	ws.Write("resetLocalState", nil)

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client, in a separate goroutine.
	go ws.HandleNotifiers(
		web.arena.MatchLoadNotifier,
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.ReloadDisplaysNotifier,
	)

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		command, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				// Client has closed the connection; nothing to do here.
				return
			}
			log.Println(err)
			return
		}
		score := &(*realtimeScore).CurrentScore
		scoreChanged := false

		if command == "commitMatch" {
			if web.arena.MatchState != field.PostMatch {
				// Don't allow committing the score until the match is over.
				ws.WriteError("Cannot commit score: Match is not over.")
				continue
			}
			web.arena.ScoringPanelRegistry.SetScoreCommitted(position, ws)
			web.arena.ScoringStatusNotifier.Notify()
		} else if command == "endgame" || command == "autoClimb" || command == "teleopClimb" {
			args := struct {
				TeamPosition  int
				EndgameStatus int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}

			if args.TeamPosition >= 1 && args.TeamPosition <= 3 && args.EndgameStatus >= 0 && args.EndgameStatus <= 3 {
				endgameStatus := game.EndgameStatus(args.EndgameStatus)
				if command == "autoClimb" {
					// Auto climb only allows Level 1 or None
					if endgameStatus == game.EndgameNone || endgameStatus == game.EndgameLevel1 {
						score.AutoClimbStatuses[args.TeamPosition-1] = endgameStatus
						scoreChanged = true
					}
				} else {
					// Default to teleop climb for "endgame" and "teleopClimb" commands
					score.TeleopClimbStatuses[args.TeamPosition-1] = endgameStatus
					scoreChanged = true
				}
			}
		} else if command == "addFoul" {
			args := struct {
				Alliance string
				IsMajor  bool
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}

			// Add the foul to the correct alliance's list.
			foul := game.Foul{FoulId: web.arena.NextFoulId, IsMajor: args.IsMajor}
			web.arena.NextFoulId++
			if args.Alliance == "red" {
				web.arena.RedRealtimeScore.CurrentScore.Fouls =
					append(web.arena.RedRealtimeScore.CurrentScore.Fouls, foul)
			} else {
				web.arena.BlueRealtimeScore.CurrentScore.Fouls =
					append(web.arena.BlueRealtimeScore.CurrentScore.Fouls, foul)
			}
			web.arena.RealtimeScoreNotifier.Notify()
		} else {
			args := struct {
				Adjustment int
				Current    bool
				Autonomous bool
				NearSide   bool
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}

			// TODO: Add REBUILT-specific scoring commands here
			switch command {
			case "activeFuel":
				oldActiveFuel := score.ActiveFuel
				score.ActiveFuel = max(0, score.ActiveFuel+args.Adjustment)
				delta := score.ActiveFuel - oldActiveFuel
				if delta != 0 {
					// Track shift-by-shift fuel for diagnostics
					// Active fuel uses grace period (hubActive=true)
					matchTimeSec := web.arena.MatchTimeSec()
					phase := game.GetMatchPhaseForTracking(matchTimeSec, true)
					trackShiftFuelForScoring(score, delta, phase, true)
					scoreChanged = true
				}
			case "inactiveFuel":
				oldInactiveFuel := score.InactiveFuel
				score.InactiveFuel = max(0, score.InactiveFuel+args.Adjustment)
				delta := score.InactiveFuel - oldInactiveFuel
				if delta != 0 {
					// Track shift-by-shift fuel for diagnostics
					// Inactive fuel uses immediate boundaries (hubActive=false)
					matchTimeSec := web.arena.MatchTimeSec()
					phase := game.GetMatchPhaseForTracking(matchTimeSec, false)
					trackShiftFuelForScoring(score, delta, phase, false)
					scoreChanged = true
				}
			case "autoFuel":
				if args.Autonomous {
					score.AutoFuel = max(0, score.AutoFuel+args.Adjustment)
					scoreChanged = true
				}
			}
		}

		if scoreChanged {
			web.arena.RealtimeScoreNotifier.Notify()
		}
	}
}

// trackShiftFuelForScoring tracks fuel scored during each match phase for diagnostic purposes.
// This is used for manual scoring when there is no PLC.
func trackShiftFuelForScoring(score *game.Score, delta int, phase game.MatchPhase, hubActive bool) {
	if hubActive {
		switch phase {
		case game.PhaseAuto, game.PhaseTransition:
			// PhaseAuto can occur during teleop grace period - count as transition
			score.TransitionFuel += delta
		case game.PhaseShift1:
			score.Shift1Fuel += delta
		case game.PhaseShift2:
			score.Shift2Fuel += delta
		case game.PhaseShift3:
			score.Shift3Fuel += delta
		case game.PhaseShift4:
			score.Shift4Fuel += delta
		case game.PhaseEndGame:
			score.EndGameFuel += delta
		}
	} else {
		switch phase {
		case game.PhaseAuto, game.PhaseTransition:
			// PhaseAuto can occur during teleop grace period - count as transition
			score.TransitionFuelInactive += delta
		case game.PhaseShift1:
			score.Shift1FuelInactive += delta
		case game.PhaseShift2:
			score.Shift2FuelInactive += delta
		case game.PhaseShift3:
			score.Shift3FuelInactive += delta
		case game.PhaseShift4:
			score.Shift4FuelInactive += delta
		case game.PhaseEndGame:
			score.EndGameFuelInactive += delta
		}
	}
}
