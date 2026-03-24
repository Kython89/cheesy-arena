// Copyright 2026 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Web handlers for the "team_sign_back" display.

package web

import (
	"io"
	"log"
	"net/http"

	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/websocket"
	"github.com/mitchellh/mapstructure"
)

// Renders the team sign back display.
func (web *Web) teamSignBackDisplayHandler(w http.ResponseWriter, r *http.Request) {
	if !web.enforceDisplayConfiguration(
		w,
		r,
		map[string]string{
			"background":    "#000",
			"alliance":      "red",
			"show_inactive": "false",
		},
	) {
		return
	}

	template, err := web.parseFiles("templates/team_sign_back_display.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	data := struct {
		*model.EventSettings
	}{web.arena.EventSettings}
	err = template.ExecuteTemplate(w, "team_sign_back_display.html", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the team sign back display client to receive status updates.
func (web *Web) teamSignBackDisplayWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	display, err := web.registerDisplay(r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer web.arena.MarkDisplayDisconnected(display.DisplayConfiguration.Id)

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer ws.Close()

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client, in a separate goroutine.
	go ws.HandleNotifiers(
		display.Notifier,
		web.arena.ArenaStatusNotifier,
		web.arena.MatchTimingNotifier,
		web.arena.AudienceDisplayModeNotifier,
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

		log.Printf("Team sign back received command: %s", command)

		switch command {
		case "startMatch":
			args := struct {
				MuteMatchSounds bool
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
			web.arena.MuteMatchSounds = args.MuteMatchSounds
			err = web.arena.StartMatch()
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
		case "abortMatch":
			err = web.arena.AbortMatch()
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
		case "commitResults":
			if web.arena.MatchState != field.PostMatch {
				ws.WriteError("cannot commit match while it is in progress")
				continue
			}
			err = web.commitCurrentMatchScore()
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
			err = web.arena.ResetMatch()
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
			err = web.arena.LoadNextMatch(false)
			if err != nil {
				ws.WriteError(err.Error())
				continue
			}
		case "discardResults":
			log.Println("Processing discardResults - calling ResetMatch")
			err = web.arena.ResetMatch()
			if err != nil {
				log.Printf("ResetMatch error: %v", err)
				ws.WriteError(err.Error())
				continue
			}
			log.Println("ResetMatch succeeded - calling LoadNextMatch")
			err = web.arena.LoadNextMatch(false)
			if err != nil {
				log.Printf("LoadNextMatch error: %v", err)
				ws.WriteError(err.Error())
				continue
			}
			log.Println("discardResults completed successfully")
		}
	}
}
