// Copyright 2024 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package field

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"image/color"
	"testing"
)

func TestTeamSign_GenerateInMatchRearText(t *testing.T) {
	realtimeScore1 := &RealtimeScore{AmplifiedTimeRemainingSec: 9}
	realtimeScore2 := &RealtimeScore{CurrentScore: game.Score{AmpSpeaker: game.AmpSpeaker{AutoSpeakerNotes: 12}}}

	assert.Equal(t, "01:23  00/18  Amp: 9", generateInMatchRearText("01:23", realtimeScore1, realtimeScore2))
	game.MelodyBonusThresholdWithoutCoop = 23
	assert.Equal(t, "34:56  12/23        ", generateInMatchRearText("34:56", realtimeScore2, realtimeScore1))
}

func TestTeamSign_Timer(t *testing.T) {
	arena := setupTestArena(t)
	sign := TeamSign{isTimer: true}

	// Should do nothing if no address is set.
	sign.update(arena, nil, true, "12:34", "Rear Text")
	assert.Equal(t, [128]byte{}, sign.packetData)

	// Check some basics about the data but don't unit-test the whole packet.
	sign.SetAddress("10.0.100.56")
	sign.update(arena, nil, true, "12:34", "Rear Text")
	assert.Equal(t, "CYPRX", string(sign.packetData[0:5]))
	assert.Equal(t, 56, int(sign.packetData[5]))
	assert.Equal(t, 0x04, int(sign.packetData[6]))
	assert.Equal(t, "12:34", string(sign.packetData[10:15]))
	assert.Equal(t, "Rear Text", string(sign.packetData[30:39]))
	assert.Equal(t, 40, sign.packetIndex)

	arena.FieldReset = false
	frontText, frontColor := generateTimerText(false, "23:45")
	assert.Equal(t, "23:45", frontText)
	assert.Equal(t, whiteColor, frontColor)
	frontText, frontColor = generateTimerText(true, "23:45")
	assert.Equal(t, "SAFE", frontText)
	assert.Equal(t, greenColor, frontColor)
}

func TestTeamSign_TeamNumber(t *testing.T) {
	arena := setupTestArena(t)
	allianceStation := arena.AllianceStations["R1"]
	arena.Database.CreateTeam(&model.Team{Id: 254})
	sign := &TeamSign{isTimer: false}

	// Should do nothing if no address is set.
	sign.update(arena, allianceStation, true, "12:34", "Rear Text")
	assert.Equal(t, [128]byte{}, sign.packetData)

	// Check some basics about the data but don't unit-test the whole packet.
	sign.SetAddress("10.0.100.53")
	sign.update(arena, allianceStation, true, "12:34", "Rear Text")
	assert.Equal(t, "CYPRX", string(sign.packetData[0:5]))
	assert.Equal(t, 53, int(sign.packetData[5]))
	assert.Equal(t, 0x04, int(sign.packetData[6]))
	assert.Equal(t, []byte{0x01, 53, 0x01, 0, 0}, sign.packetData[7:12])
	assert.Equal(t, "No Team Assigned", string(sign.packetData[29:45]))
	assert.Equal(t, 46, sign.packetIndex)

	assertSign := func(isRed bool, expectedFrontText string, expectedFrontColor color.RGBA, expectedRearText string) {
		frontText, frontColor, rearText := generateTeamNumberTexts(arena, allianceStation, isRed, "Rear Text")
		assert.Equal(t, expectedFrontText, frontText)
		assert.Equal(t, expectedFrontColor, frontColor)
		assert.Equal(t, expectedRearText, rearText)
	}

	assertSign(true, "", whiteColor, "    No Team Assigned")
	arena.FieldReset = true
	arena.assignTeam(254, "R1")
	assertSign(true, "  254", greenColor, "254       Connect PC")
	assertSign(false, "  254", greenColor, "254       Connect PC")
	arena.FieldReset = false
	assertSign(true, "  254", redColor, "254       Connect PC")
	assertSign(false, "  254", blueColor, "254       Connect PC")

	// Check through pre-match sequence.
	allianceStation.Ethernet = true
	assertSign(true, "  254", redColor, "254         Start DS")
	allianceStation.DsConn = &DriverStationConnection{}
	assertSign(true, "  254", redColor, "254         No Radio")
	allianceStation.DsConn.WrongStation = "R1"
	assertSign(true, "  254", redColor, "254     Move Station")
	allianceStation.DsConn.WrongStation = ""
	allianceStation.DsConn.RadioLinked = true
	assertSign(true, "  254", redColor, "254           No Rio")
	allianceStation.DsConn.RioLinked = true
	assertSign(true, "  254", redColor, "254          No Code")
	allianceStation.DsConn.RobotLinked = true
	assertSign(true, "  254", redColor, "254            Ready")
	allianceStation.Bypass = true
	assertSign(true, "  254", redColor, "254         Bypassed")

	// Check E-stop and A-stop.
	arena.MatchState = AutoPeriod
	assertSign(true, "  254", redColor, "Rear Text")
	allianceStation.AStop = true
	assertSign(true, "  254", orangeColor, "254           A-STOP")
	allianceStation.EStop = true
	assertSign(false, "  254", orangeColor, "254           E-STOP")
	allianceStation.EStop = false
	arena.MatchState = TeleopPeriod
	assertSign(false, "  254", blueColor, "Rear Text")
	allianceStation.EStop = true
	assertSign(false, "  254", orangeColor, "254           E-STOP")
	arena.MatchState = PostMatch
	assertSign(false, "  254", orangeColor, "254           E-STOP")
}
