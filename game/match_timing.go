// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Game-specific period timing.

package game

import "time"

const (
	TeleopGracePeriodSec     = 3
	TransitionDurationSec    = 10 // First 10 seconds of teleop when both hubs are active (transition period)
	ShiftDurationSec         = 25
	HubScoringGracePeriodSec = 3  // Grace period after hub deactivates to still count FUEL as active
	EndGameDurationSec       = 30 // Last 30 seconds of teleop when both hubs are active
)

var MatchTiming = struct {
	WarmupDurationSec           int
	AutoDurationSec             int
	PauseDurationSec            int
	TeleopDurationSec           int
	WarningRemainingDurationSec int
	TimeoutDurationSec          int
}{0, 20, 3, 140, 20, 0}

func GetDurationToAutoEnd() time.Duration {
	return time.Duration(MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec) * time.Second
}

func GetDurationToTeleopStart() time.Duration {
	return time.Duration(
		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.PauseDurationSec,
	) * time.Second
}

func GetDurationToTeleopEnd() time.Duration {
	return time.Duration(
		MatchTiming.WarmupDurationSec+MatchTiming.AutoDurationSec+MatchTiming.PauseDurationSec+
			MatchTiming.TeleopDurationSec,
	) * time.Second
}

// GetCurrentShift returns which shift number (0-indexed) is currently active based on match time.
// Returns -1 if not in teleop period.
func GetCurrentShift(matchTimeSec float64) int {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)

	if matchTimeSec < teleopStartSec || matchTimeSec >= teleopEndSec {
		return -1
	}

	teleopElapsedSec := matchTimeSec - teleopStartSec
	return int(teleopElapsedSec / ShiftDurationSec)
}

// MatchPhase represents the current phase of the match for diagnostic tracking
type MatchPhase int

const (
	PhaseAuto       MatchPhase = iota // Auto period
	PhaseTransition                   // First 10 seconds of teleop
	PhaseShift1                       // First alternating shift (10-35 sec into teleop)
	PhaseShift2                       // Second alternating shift (35-60 sec into teleop)
	PhaseShift3                       // Third alternating shift (60-85 sec into teleop)
	PhaseShift4                       // Fourth alternating shift (85-110 sec into teleop)
	PhaseEndGame                      // Last 30 seconds of teleop (110-140 sec into teleop)
	PhaseNone                         // Not in a scoring period
)

// GetMatchPhase returns the current match phase for diagnostic tracking.
// This is a convenience wrapper that uses grace period (for active fuel tracking).
func GetMatchPhase(matchTimeSec float64) MatchPhase {
	return GetMatchPhaseForTracking(matchTimeSec, true)
}

// GetMatchPhaseForTracking returns the current match phase for diagnostic tracking.
// The hubActive parameter determines the phase boundaries:
// - hubActive=true (active fuel): Phase starts immediately, extends 3 sec past end (25+3=28 sec window)
// - hubActive=false (inactive fuel): Phase starts 3 sec after boundary, ends immediately (25-3=22 sec window)
func GetMatchPhaseForTracking(matchTimeSec float64, hubActive bool) MatchPhase {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)
	transitionEndSec := teleopStartSec + float64(TransitionDurationSec)
	endGameStartSec := teleopEndSec - float64(EndGameDurationSec)
	gracePeriodSec := float64(HubScoringGracePeriodSec)

	// Before teleop starts (auto or pause)
	if matchTimeSec < teleopStartSec {
		return PhaseAuto
	}

	// After match ends
	if matchTimeSec >= teleopEndSec {
		return PhaseNone
	}

	if hubActive {
		// ACTIVE fuel tracking:
		// - Phase starts immediately at boundary
		// - Phase extends 3 seconds past the end (grace period for balls in flight)

		// Transition: 0 to transitionEnd+3
		if matchTimeSec < transitionEndSec+gracePeriodSec {
			return PhaseTransition
		}

		// End game starts at endGameStartSec, but Shift 4 grace extends into it
		// If we're in Shift 4 grace period, still count as Shift 4
		if matchTimeSec >= endGameStartSec && matchTimeSec < endGameStartSec+gracePeriodSec {
			return PhaseShift4
		}

		// End game (no grace needed at match end)
		if matchTimeSec >= endGameStartSec {
			return PhaseEndGame
		}

		// Alternating shifts - check if we're in a shift's grace period
		postTransitionSec := matchTimeSec - transitionEndSec
		shift := int(postTransitionSec / ShiftDurationSec)
		shiftEndSec := transitionEndSec + float64(shift+1)*ShiftDurationSec

		// If we're past the shift end but within grace, still count as this shift
		if matchTimeSec >= shiftEndSec && matchTimeSec < shiftEndSec+gracePeriodSec {
			// Still in grace period of current shift
		} else if matchTimeSec >= shiftEndSec {
			// Past grace period, move to next shift
			shift++
		}

		switch shift {
		case 0:
			return PhaseShift1
		case 1:
			return PhaseShift2
		case 2:
			return PhaseShift3
		case 3:
			return PhaseShift4
		default:
			return PhaseEndGame
		}
	} else {
		// INACTIVE fuel tracking:
		// - Phase starts 3 seconds after boundary (after previous active grace ends)
		// - Phase ends immediately at the next boundary

		// Transition grace period - inactive scoring hasn't started yet
		if matchTimeSec < transitionEndSec+gracePeriodSec {
			return PhaseTransition // Still in transition grace, no inactive shift yet
		}

		// End game: inactive scoring ends immediately when end game starts
		if matchTimeSec >= endGameStartSec {
			return PhaseEndGame
		}

		// Alternating shifts with delayed start
		// Inactive shift N starts at (shiftN_start + 3) and ends at shiftN_end
		postTransitionSec := matchTimeSec - transitionEndSec
		shift := int(postTransitionSec / ShiftDurationSec)
		shiftStartSec := transitionEndSec + float64(shift)*ShiftDurationSec

		// If we're in the first 3 seconds of this shift, we're still in the previous shift's grace
		// For inactive, this means we count towards the previous inactive shift (or transition)
		if matchTimeSec < shiftStartSec+gracePeriodSec {
			if shift == 0 {
				return PhaseTransition
			}
			shift--
		}

		switch shift {
		case 0:
			return PhaseShift1
		case 1:
			return PhaseShift2
		case 2:
			return PhaseShift3
		case 3:
			return PhaseShift4
		default:
			return PhaseEndGame
		}
	}
}

// IsRedHubActive returns true if the red alliance's hub is currently active.
// During auto and pause, both hubs are active.
// During the first 10 seconds of teleop (transition period), both hubs are active.
// During the last 30 seconds of teleop (END GAME), both hubs are active.
// After the transition, the alliance that LOST auto has their hub active first.
// If Red won auto: Red is INACTIVE first, then alternates every 25 seconds.
// If Blue won auto or tie: Red is ACTIVE first, then alternates every 25 seconds.
func IsRedHubActive(matchTimeSec float64, redWonAuto bool) bool {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)
	transitionEndSec := teleopStartSec + float64(TransitionDurationSec)

	// During auto and pause, both hubs are active
	if matchTimeSec < teleopStartSec {
		return true
	}

	// During transition period (first 10 seconds of teleop), both hubs are active
	if matchTimeSec < transitionEndSec {
		return true
	}

	// During END GAME (last 30 seconds of teleop), both hubs are active
	if matchTimeSec >= teleopEndSec-EndGameDurationSec && matchTimeSec < teleopEndSec {
		return true
	}

	// After the match ends, hubs are not active
	if matchTimeSec >= teleopEndSec {
		return false
	}

	// Calculate which alternating shift we're in (after transition period)
	// Shift 0 = 10-35 sec into teleop, Shift 1 = 35-60 sec, etc.
	postTransitionSec := matchTimeSec - transitionEndSec
	if postTransitionSec < 0 {
		return false
	}
	shift := int(postTransitionSec / ShiftDurationSec)

	if redWonAuto {
		// Red won auto, so Red hub is INACTIVE for first alternating shift
		// Red is INACTIVE on even shifts (0, 2, 4...), ACTIVE on odd shifts (1, 3, 5...)
		return shift%2 == 1
	} else {
		// Blue won auto or tie, so Red hub is ACTIVE for first alternating shift
		// Red is ACTIVE on even shifts (0, 2, 4...), INACTIVE on odd shifts (1, 3, 5...)
		return shift%2 == 0
	}
}

// IsBlueHubActive returns true if the blue alliance's hub is currently active.
// During auto and pause, both hubs are active.
// During the first 10 seconds of teleop (transition period), both hubs are active.
// During the last 30 seconds of teleop (END GAME), both hubs are active.
// After the transition, the alliance that LOST auto has their hub active first.
// If Blue won auto: Blue is INACTIVE first, then alternates every 25 seconds.
// If Red won auto or tie: Blue is ACTIVE first, then alternates every 25 seconds.
func IsBlueHubActive(matchTimeSec float64, blueWonAuto bool) bool {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)
	transitionEndSec := teleopStartSec + float64(TransitionDurationSec)

	// During auto and pause, both hubs are active
	if matchTimeSec < teleopStartSec {
		return true
	}

	// During transition period (first 10 seconds of teleop), both hubs are active
	if matchTimeSec < transitionEndSec {
		return true
	}

	// During END GAME (last 30 seconds of teleop), both hubs are active
	if matchTimeSec >= teleopEndSec-EndGameDurationSec && matchTimeSec < teleopEndSec {
		return true
	}

	// After the match ends, hubs are not active
	if matchTimeSec >= teleopEndSec {
		return false
	}

	// Calculate which alternating shift we're in (after transition period)
	// Shift 0 = 10-35 sec into teleop, Shift 1 = 35-60 sec, etc.
	postTransitionSec := matchTimeSec - transitionEndSec
	if postTransitionSec < 0 {
		return false
	}
	shift := int(postTransitionSec / ShiftDurationSec)

	if blueWonAuto {
		// Blue won auto, so Blue hub is INACTIVE for first alternating shift
		// Blue is INACTIVE on even shifts (0, 2, 4...), ACTIVE on odd shifts (1, 3, 5...)
		return shift%2 == 1
	} else {
		// Red won auto or tie, so Blue hub is ACTIVE for first alternating shift
		// Blue is ACTIVE on even shifts (0, 2, 4...), INACTIVE on odd shifts (1, 3, 5...)
		return shift%2 == 0
	}
}

// IsRedHubActiveForScoring returns true if the red alliance's hub should accept FUEL as "active".
// This includes the grace period after the hub deactivates to account for FUEL in flight.
// The grace period applies even after the match ends.
func IsRedHubActiveForScoring(matchTimeSec float64, redWonAuto bool) bool {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)
	transitionEndSec := teleopStartSec + float64(TransitionDurationSec)

	// Check if hub is currently active
	if IsRedHubActive(matchTimeSec, redWonAuto) {
		return true
	}

	// Check if we're in the grace period after the match ends
	if matchTimeSec >= teleopEndSec && matchTimeSec < teleopEndSec+HubScoringGracePeriodSec {
		// Both hubs are active during END GAME, so grace period applies
		return true
	}

	// Check if we're in the grace period after the transition period ends
	if matchTimeSec >= transitionEndSec && matchTimeSec < transitionEndSec+HubScoringGracePeriodSec {
		// Both hubs are active during transition, so grace period applies
		return true
	}

	// Calculate which alternating shift we're in (after transition period)
	postTransitionSec := matchTimeSec - transitionEndSec
	if postTransitionSec < 0 {
		return false
	}
	shift := int(postTransitionSec / ShiftDurationSec)
	timeInShift := postTransitionSec - float64(shift)*ShiftDurationSec

	// If we're within the grace period after a shift transition, check if the hub was active in the previous moment
	if timeInShift < HubScoringGracePeriodSec {
		// Check if the hub was active at the end of the previous shift
		previousShift := shift - 1
		if previousShift < 0 {
			// Previous shift was the transition period, which had both hubs active
			return true
		}

		if redWonAuto {
			// Red won auto, so Red is INACTIVE on even shifts, ACTIVE on odd shifts
			return previousShift%2 == 1
		} else {
			// Blue won auto or tie, so Red is ACTIVE on even shifts, INACTIVE on odd shifts
			return previousShift%2 == 0
		}
	}

	return false
}

// IsBlueHubActiveForScoring returns true if the blue alliance's hub should accept FUEL as "active".
// This includes the grace period after the hub deactivates to account for FUEL in flight.
// The grace period applies even after the match ends.
func IsBlueHubActiveForScoring(matchTimeSec float64, blueWonAuto bool) bool {
	teleopStartSec := float64(MatchTiming.WarmupDurationSec + MatchTiming.AutoDurationSec + MatchTiming.PauseDurationSec)
	teleopEndSec := teleopStartSec + float64(MatchTiming.TeleopDurationSec)
	transitionEndSec := teleopStartSec + float64(TransitionDurationSec)

	// Check if hub is currently active
	if IsBlueHubActive(matchTimeSec, blueWonAuto) {
		return true
	}

	// Check if we're in the grace period after the match ends
	if matchTimeSec >= teleopEndSec && matchTimeSec < teleopEndSec+HubScoringGracePeriodSec {
		// Both hubs are active during END GAME, so grace period applies
		return true
	}

	// Check if we're in the grace period after the transition period ends
	if matchTimeSec >= transitionEndSec && matchTimeSec < transitionEndSec+HubScoringGracePeriodSec {
		// Both hubs are active during transition, so grace period applies
		return true
	}

	// Calculate which alternating shift we're in (after transition period)
	postTransitionSec := matchTimeSec - transitionEndSec
	if postTransitionSec < 0 {
		return false
	}
	shift := int(postTransitionSec / ShiftDurationSec)
	timeInShift := postTransitionSec - float64(shift)*ShiftDurationSec

	// If we're within the grace period after a shift transition, check if the hub was active in the previous moment
	if timeInShift < HubScoringGracePeriodSec {
		// Check if the hub was active at the end of the previous shift
		previousShift := shift - 1
		if previousShift < 0 {
			// Previous shift was the transition period, which had both hubs active
			return true
		}

		if blueWonAuto {
			// Blue won auto, so Blue is INACTIVE on even shifts, ACTIVE on odd shifts
			return previousShift%2 == 1
		} else {
			// Red won auto or tie, so Blue is ACTIVE on even shifts, INACTIVE on odd shifts
			return previousShift%2 == 0
		}
	}

	return false
}
