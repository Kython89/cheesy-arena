// Copyright 2026 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side methods for the "team_sign_back" display.

var websocket;
var currentMatch;
var currentMatchTimeData = null;
var currentScoreData = null;
var displayAlliance = "Red"; // Which alliance's data to show, set from URL param
var showInactive = false; // If true, show inactive fuel count instead of auto climb points

// Game timing constants (must match game/match_timing.go)
const transitionDurationSec = 10; // First 10 seconds of teleop when both hubs are active
const shiftDurationSec = 25;
const endGameDurationSec = 30; // Last 30 seconds of teleop when both hubs are active

// RP thresholds (must match game/score.go)
const energizedRPThreshold = 100;
const superchargedRPThreshold = 360;

// Handles a websocket message to change which screen is displayed.
const handleAudienceDisplayMode = function (targetScreen) {
  // For this display, we mostly care about match state which comes from matchTime.
};

// Handles a websocket message to update the teams for the current match.
const handleMatchLoad = function (data) {
  currentMatch = data.Match;
  updateDisplay();
};

// Handles a websocket message to update the match time countdown.
const handleMatchTime = function (data) {
  currentMatchTimeData = data;
  updateDisplay();
};

// Handles a websocket message to update the match score.
const handleRealtimeScore = function (data) {
  currentScoreData = data;
  updateDisplay();
};

// Determines if red won auto based on current score data.
const didRedWinAuto = function () {
  if (!currentScoreData) {
    return false;
  }
  const redAutoPoints = currentScoreData.Red.ScoreSummary.AutoPoints;
  const blueAutoPoints = currentScoreData.Blue.ScoreSummary.AutoPoints;
  if (redAutoPoints > blueAutoPoints) {
    return true;
  }
  if (redAutoPoints === blueAutoPoints && currentScoreData.AutoTieWinner === "red") {
    return true;
  }
  return false;
};

// Returns shift info: { shiftChar, timeLeft, activeAlliance }
// shiftChar: A=Auto, T=Transition, R=Red active, B=Blue active, E=End game
// activeAlliance: "Red", "Blue", or "Both"
const getShiftInfo = function () {
  if (!matchTiming || !currentMatchTimeData) {
    return { shiftChar: "P", timeLeft: 0, activeAlliance: "Both" };
  }

  const matchTimeSec = currentMatchTimeData.MatchTimeSec;
  const matchState = currentMatchTimeData.MatchState;

  // Calculate key time boundaries using matchTiming from websocket
  const warmupDurationSec = matchTiming.WarmupDurationSec;
  const autoDurationSec = matchTiming.AutoDurationSec;
  const pauseDurationSec = matchTiming.PauseDurationSec;
  const teleopDurationSec = matchTiming.TeleopDurationSec;

  const autoEndSec = warmupDurationSec + autoDurationSec;
  const teleopStartSec = autoEndSec + pauseDurationSec;
  const transitionEndSec = teleopStartSec + transitionDurationSec;
  const teleopEndSec = teleopStartSec + teleopDurationSec;
  const endGameStartSec = teleopEndSec - endGameDurationSec;

  // PRE_MATCH, START_MATCH, WARMUP_PERIOD
  if (matchState === 0 || matchState === 1 || matchState === 2) {
    return { shiftChar: "P", timeLeft: 0, activeAlliance: "Both" };
  }

  // AUTO_PERIOD - both hubs active
  if (matchState === 3) {
    const timeLeft = Math.max(0, Math.ceil(autoEndSec - matchTimeSec));
    return { shiftChar: "A", timeLeft: timeLeft, activeAlliance: "Both" };
  }

  // PAUSE_PERIOD - both hubs active, show T for transition coming
  if (matchState === 4) {
    const timeLeft = Math.max(0, Math.ceil(transitionEndSec - matchTimeSec));
    return { shiftChar: "T", timeLeft: timeLeft, activeAlliance: "Both" };
  }

  // TELEOP_PERIOD
  if (matchState === 5) {
    // Transition period (first 10 seconds of teleop) - both hubs active
    if (matchTimeSec < transitionEndSec) {
      const timeLeft = Math.max(0, Math.ceil(transitionEndSec - matchTimeSec));
      return { shiftChar: "T", timeLeft: timeLeft, activeAlliance: "Both" };
    }

    // End game (last 30 seconds) - both hubs active
    if (matchTimeSec >= endGameStartSec) {
      const timeLeft = Math.max(0, Math.ceil(teleopEndSec - matchTimeSec));
      return { shiftChar: "E", timeLeft: timeLeft, activeAlliance: "Both" };
    }

    // Alternating shifts between transition end and end game start
    const postTransitionSec = matchTimeSec - transitionEndSec;
    const shift = Math.floor(postTransitionSec / shiftDurationSec);
    const timeInShift = postTransitionSec - (shift * shiftDurationSec);
    const timeLeft = Math.max(0, Math.ceil(shiftDurationSec - timeInShift));

    // Determine which hub is active
    // The alliance that LOST auto has their hub active first
    const redWonAuto = didRedWinAuto();
    let redHubActive;
    if (redWonAuto) {
      // Red won auto, so Red hub is INACTIVE first (even shifts), ACTIVE on odd shifts
      redHubActive = (shift % 2 === 1);
    } else {
      // Blue won auto or tie, so Red hub is ACTIVE first (even shifts), INACTIVE on odd shifts
      redHubActive = (shift % 2 === 0);
    }

    if (redHubActive) {
      return { shiftChar: "R", timeLeft: timeLeft, activeAlliance: "Red" };
    } else {
      return { shiftChar: "B", timeLeft: timeLeft, activeAlliance: "Blue" };
    }
  }

  // POST_MATCH
  if (matchState === 6) {
    return { shiftChar: "E", timeLeft: 0, activeAlliance: "Both" };
  }

  return { shiftChar: "P", timeLeft: 0, activeAlliance: "Both" };
};

// Determines which shifts are active for the display alliance based on who won auto
const getActiveShifts = function () {
  const redWonAuto = didRedWinAuto();
  const isRed = (displayAlliance === "Red");

  // The alliance that LOST auto has their hub active first (shifts 1 and 3)
  // The alliance that WON auto has their hub active second (shifts 2 and 4)
  if (isRed) {
    if (redWonAuto) {
      // Red won, so Red is INACTIVE first -> Red active on shifts 2 and 4
      return { shift1: false, shift2: true, shift3: false, shift4: true };
    } else {
      // Red lost or tie, so Red is ACTIVE first -> Red active on shifts 1 and 3
      return { shift1: true, shift2: false, shift3: true, shift4: false };
    }
  } else {
    // Blue alliance
    if (redWonAuto) {
      // Red won, so Blue lost -> Blue is ACTIVE first -> Blue active on shifts 1 and 3
      return { shift1: true, shift2: false, shift3: true, shift4: false };
    } else {
      // Blue won or tie, so Blue is INACTIVE first -> Blue active on shifts 2 and 4
      return { shift1: false, shift2: true, shift3: false, shift4: true };
    }
  }
};

const updateDisplay = function () {
  if (!currentMatch) {
    $("#displayLine1").text("WAITING");
    $("#displayLine2").hide();
    $("#displayLine3").hide();
    return;
  }

  // Only show during Test, Practice and Qualification matches (not Playoff)
  if (currentMatch.Type === matchTypePlayoff) {
    $("#displayLine1").text("QUAL ONLY");
    $("#displayLine2").hide();
    $("#displayLine3").hide();
    return;
  }

  if (!currentMatchTimeData || !matchTiming) {
    $("#displayLine1").text("P00 0/100 0 0:00");
    $("#displayLine2").hide();
    $("#displayLine3").hide();
    return;
  }

  $("#displayLine2").show();
  $("#displayLine3").show();

  // 1. Current SHIFT and time remaining in that period
  const shiftInfo = getShiftInfo();
  const timeLeftStr = shiftInfo.timeLeft < 10 ? "0" + shiftInfo.timeLeft : String(shiftInfo.timeLeft);
  const shiftDisplay = shiftInfo.shiftChar + timeLeftStr;

  // 2. Progress towards the FUEL Ranking Points
  // Always show the alliance specified by the URL param
  // Only AutoFuel + ActiveFuel count towards RP (not InactiveFuel)
  let fuelForRP = 0;
  let fuelThreshold = energizedRPThreshold;
  let thirdFieldValue = 0;

  // Shift-by-shift counts
  let autoCount = 0;
  let transitionCount = 0;
  let endGameCount = 0;
  let shift1Count = 0;
  let shift2Count = 0;
  let shift3Count = 0;
  let shift4Count = 0;

  if (currentScoreData) {
    const score = currentScoreData[displayAlliance].Score;
    fuelForRP = score.AutoFuel + score.ActiveFuel;
    // Once we pass energized threshold, show progress towards supercharged
    if (fuelForRP >= energizedRPThreshold) {
      fuelThreshold = superchargedRPThreshold;
    }

    if (showInactive) {
      // Show inactive fuel count
      thirdFieldValue = score.InactiveFuel;
    } else {
      // Show auto climb points
      thirdFieldValue = currentScoreData[displayAlliance].ScoreSummary.AutoClimbPoints;
    }

    // Get shift-by-shift counts (active + inactive for each phase)
    autoCount = score.AutoFuel || 0;
    transitionCount = (score.TransitionFuel || 0) + (score.TransitionFuelInactive || 0);
    endGameCount = (score.EndGameFuel || 0) + (score.EndGameFuelInactive || 0);
    shift1Count = (score.Shift1Fuel || 0) + (score.Shift1FuelInactive || 0);
    shift2Count = (score.Shift2Fuel || 0) + (score.Shift2FuelInactive || 0);
    shift3Count = (score.Shift3Fuel || 0) + (score.Shift3FuelInactive || 0);
    shift4Count = (score.Shift4Fuel || 0) + (score.Shift4FuelInactive || 0);
  }

  const rpProgress = fuelForRP + "/" + fuelThreshold;

  // 3. Either AUTO TOWER points or inactive fuel count depending on show_inactive param
  const thirdFieldDisplay = String(thirdFieldValue);

  // 4. Remaining MATCH period time
  let matchTimeRemaining = "0:00";
  translateMatchTime(currentMatchTimeData, function (state, stateText, countdownSec) {
    matchTimeRemaining = getCountdownString(countdownSec);
  });

  // Update line 1 (main display)
  const fullText = shiftDisplay + " " + rpProgress + " " + thirdFieldDisplay + " " + matchTimeRemaining;
  $("#displayLine1").text(fullText);

  // Update line 2 (A, T, E counts) - always alliance color since these are always active
  $("#autoCount").text(autoCount);
  $("#transitionCount").text(transitionCount);
  $("#endGameCount").text(endGameCount);

  // Update line 3 (shift counts) with active/inactive coloring
  const activeShifts = getActiveShifts();
  const allianceColor = (displayAlliance === "Blue") ? "#00f" : "#f00";

  $("#shift1Count").text(shift1Count).css("color", activeShifts.shift1 ? allianceColor : "#fff");
  $("#shift2Count").text(shift2Count).css("color", activeShifts.shift2 ? allianceColor : "#fff");
  $("#shift3Count").text(shift3Count).css("color", activeShifts.shift3 ? allianceColor : "#fff");
  $("#shift4Count").text(shift4Count).css("color", activeShifts.shift4 ? allianceColor : "#fff");
};

$(function () {
  // Read the configuration for this display from the URL query string.
  const urlParams = new URLSearchParams(window.location.search);
  document.body.style.backgroundColor = urlParams.get("background");

  // Set which alliance's data to display (default to Red)
  const allianceParam = urlParams.get("alliance");
  let allianceColor;
  if (allianceParam && allianceParam.toLowerCase() === "blue") {
    displayAlliance = "Blue";
    allianceColor = "#00f";
  } else {
    displayAlliance = "Red";
    allianceColor = "#f00";
  }

  // Set colors for main display line and A/T/E counts (always alliance color)
  $("#displayLine1").css("color", allianceColor);
  $("#displayLine2").css("color", allianceColor);
  // Line 3 shift colors are set dynamically in updateDisplay based on active/inactive

  // Set whether to show inactive fuel instead of auto climb points
  const showInactiveParam = urlParams.get("show_inactive");
  showInactive = (showInactiveParam === "true");

  updateDisplay();

  websocket = new CheesyWebsocket("/displays/team_sign_back/websocket", {
    audienceDisplayMode: function (event) {
      handleAudienceDisplayMode(event.data);
    },
    matchLoad: function (event) {
      handleMatchLoad(event.data);
    },
    matchTime: function (event) {
      handleMatchTime(event.data);
    },
    matchTiming: function (event) {
      handleMatchTiming(event.data);
    },
    realtimeScore: function (event) {
      handleRealtimeScore(event.data);
    }
  });
});
