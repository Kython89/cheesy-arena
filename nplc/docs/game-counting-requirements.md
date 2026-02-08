## Match Phases, HUB Counting, and HUB Light Requirements

**Concise FMS implementation summary**

---

## 1. Match Timeline (Arena Timer)

| Phase            | Timer Range | Duration | HUB State   |
| ---------------- | ----------- | -------- | ----------- |
| AUTO             | 0:20 → 0:00 | 20 s     | Both active |
| TRANSITION SHIFT | 2:20 → 2:10 | 10 s     | Both active |
| SHIFT 1          | 2:10 → 1:45 | 25 s     | One active  |
| SHIFT 2          | 1:45 → 1:20 | 25 s     | One active  |
| SHIFT 3          | 1:20 → 0:55 | 25 s     | One active  |
| SHIFT 4          | 0:55 → 0:30 | 25 s     | One active  |
| END GAME         | 0:30 → 0:00 | 30 s     | Both active |

During the match, a HUB is either **active** or **inactive**. Only FUEL scored in an active HUB earns points. 

---

## 2. HUB Activation Logic

### Both HUBs Active

Counting enabled for both alliances during:

* AUTO
* TRANSITION SHIFT
* END GAME 

### Single HUB Active (Alliance Shifts)

During SHIFT 1–4:

* Exactly one HUB active.
* Active HUB alternates every shift boundary. 

---

## 3. Determining SHIFT 1 Active HUB

At TELEOP start:

* Alliance scoring more AUTO FUEL:

  * Their HUB is **inactive** during SHIFT 1.
* Opponent HUB is active.
* Tie → FMS randomly selects alliance.
* Status alternates each subsequent shift.
* Both HUBs return to active at END GAME start. 

FMS also publishes this result to operator consoles at TELEOP start. 

---

## 4. Counting Windows (Timing Requirements)

### A. End-of-Period Grace

FUEL scoring continues to be assessed for up to **+3.0 s** after:

* AUTO timer reaches 0:00
* Match timer reaches 0:00 

### B. HUB Deactivation Grace

When a HUB transitions **active → inactive**:

* Continue accepting scored FUEL for **+3.0 s**.
* Counts toward the previous active period. 

No grace when activating.

---

## 5. HUB Lighting Requirements (Field State Indicators)

HUB light bars indicate HUB status and upcoming state changes.

### During MATCH

| Light State                       | Meaning               | Counting Relation                                                   |
| --------------------------------- | --------------------- | ------------------------------------------------------------------- |
| Alliance color at 100% brightness | HUB active            | Counting enabled                                                    |
| Alliance color pulsing            | Deactivation warning  | Starts **3 s before deactivation**, counting still active           |
| Alliance color + white chase      | TRANSITION SHIFT only | Indicates which HUB will be inactive in SHIFT 1; HUB remains active |

 

Implementation implication:

* Pulsing state aligns with the 3 s pre-deactivation period.
* HUB remains active and scoring during the warning period.
* Actual deactivation occurs at the shift boundary.

---

### Non-Match States (Field Control)

| Light State | Meaning                      |
| ----------- | ---------------------------- |
| Off         | Match ready / HUB not active |
| Purple      | Field safe for field staff   |
| Green       | Field safe for all           |



(Not part of scoring logic but required for full field state consistency.)

---

## 6. Required FMS Logic Model (Minimal)

* Timer-driven phase state machine.
* HUB state machine:

  * Both active in AUTO / TRANSITION / END GAME.
  * One active in SHIFT 1–4.
  * Alternating active HUB each shift.
* SHIFT 1 determined by AUTO result or random selection.
* Score only while HUB active, including:

  * +3 s post-deactivation window.
  * +3 s after AUTO and match end.
* HUB lighting must reflect:

  * Active state,
  * 3 s deactivation warning,
  * SHIFT 1 inactive indication during TRANSITION SHIFT.

This defines all scoring-valid and visual state transitions required for HUB counting and field synchronization.