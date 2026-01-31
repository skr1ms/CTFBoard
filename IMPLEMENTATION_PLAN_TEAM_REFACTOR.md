# Implementation Plan: Explicit Team Selection Architecture

## Overview

Refactor team participation logic to implement explicit team selection with three distinct user states: NoTeam, Solo, and InTeam. Remove automatic team creation during registration and add proper email verification middleware.

---

## Phase 1: Database Schema Changes

### 1.1 Add new columns to `teams` table

**Migration:** `000016_add_team_participation_fields.up.sql`

```sql
ALTER TABLE teams ADD COLUMN is_solo BOOLEAN DEFAULT FALSE;
ALTER TABLE teams ADD COLUMN is_auto_created BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_teams_is_solo ON teams(is_solo);
```

### 1.2 Add new columns to `competition` table

**Migration:** `000017_add_competition_mode.up.sql`

```sql
ALTER TABLE competition ADD COLUMN mode VARCHAR(20) DEFAULT 'flexible';
ALTER TABLE competition ADD COLUMN allow_team_switch BOOLEAN DEFAULT TRUE;
ALTER TABLE competition ADD COLUMN min_team_size INT DEFAULT 1;
ALTER TABLE competition ADD COLUMN max_team_size INT DEFAULT 10;
```

### 1.3 Mark existing solo teams

**Migration:** `000018_mark_existing_solo_teams.up.sql`

```sql
UPDATE teams 
SET is_solo = TRUE, is_auto_created = TRUE
WHERE id IN (
    SELECT t.id 
    FROM teams t
    JOIN users u ON u.team_id = t.id
    GROUP BY t.id 
    HAVING COUNT(*) = 1 AND t.name = MAX(u.username)
);
```

---

## Phase 2: Entity Updates

### 2.1 Update `entity/team.go`

Add new fields:
- `IsSolo bool`
- `IsAutoCreated bool`

### 2.2 Update `entity/competition.go`

Add new fields:
- `Mode string` (solo_only, teams_only, flexible)
- `AllowTeamSwitch bool`
- `MinTeamSize int`
- `MaxTeamSize int`

### 2.3 Create `entity/participation.go`

New constants and types:
- `UserParticipationStatus` type
- `StatusNoTeam`, `StatusSolo`, `StatusInTeam` constants
- Helper methods for status determination

---

## Phase 3: Repository Layer

### 3.1 Update `repo/team_repository.go`

Add methods:
- `GetSoloTeamByUserID(ctx, userID) (*Team, error)`
- `DeleteEmptySoloTeam(ctx, tx, teamID) error`
- `CountTeamMembers(ctx, teamID) (int, error)`

### 3.2 Update `repo/tx_repository.go`

Add transaction methods:
- `CreateSoloTeamTx(ctx, tx, team) error`
- `DeleteSoloTeamTx(ctx, tx, teamID) error`
- `TransferTeamMembershipTx(ctx, tx, userID, oldTeamID, newTeamID) error`

---

## Phase 4: Use Case Layer

### 4.1 Update `usecase/user.go`

**Remove** automatic team creation from `Register()`:
- Delete team creation logic (lines 74-95)
- User should have `TeamId = nil` after registration

### 4.2 Update `usecase/team.go`

**Modify** `Create()`:
- Add `isSolo bool` parameter
- Set `team.IsSolo = isSolo`
- Handle solo team deletion logic

**Modify** `Join()`:
- Check if user has solo team
- Delete solo team before joining
- Add audit log for team transitions

**Add** new method `CreateSoloTeam(ctx, userID)`:
- Create team with `name = username`
- Set `IsSolo = true`
- Set user's `TeamId`

**Add** new method `TransitionToTeam(ctx, userID, targetTeamID)`:
- Validate transition rules
- Delete old solo team if exists
- **Delete** all solves associated with the old solo team (Option A: Clean start)
- Update user's team membership
- Create audit logs

### 4.3 Update `usecase/solve.go`

**Modify** `Submit()`:
- Check if user has team before allowing submission
- If `TeamID == nil`, return `ErrNoTeamSelected` (Strict Explicit Selection)
- Do NOT auto-create team


### 4.4 Update `usecase/hint.go`

**Modify** `UnlockHint()`:
- Check if user has team
- Return error if no team

---

### 4.5 Additional Team Management Features

**Add** to `usecase/team.go`:

*   **DisbandTeam(ctx, userId)**:
    *   Verify user is captain
    *   Delete team and all members' team associations (soft delete team, set members' team_id to nil)
    *   Create audit log
*   **KickMember(ctx, captainId, targetUserId)**:
    *   Verify requestor is captain
    *   Verify target is in the team
    *   Verify target is NOT captain
    *   Remove target from team (set team_id to nil)
    *   Create audit log

**Modify** `TransitionToTeam` and `Join`:
*   Add `confirmReset bool` parameter
*   If transitioning from Solo/Auto logic requires deleting progress AND `confirmReset` is false, return `ErrConfirmationRequired`

---

## Phase 5: Middleware

### 5.1 Create `middleware/require_team.go`

New middleware:
- Check if user has `team_id != nil`
- Check competition mode
- Allow bypass for admins
- Return 403 with proper error message

### 5.2 Create `middleware/require_verified.go`

New middleware:
- Check if email verification is enabled in config
- Check if user is verified
- Allow bypass for admins
- Return 403 with proper error message

---

## Phase 6: API Layer

### 6.1 Create `controller/restapi/v1/participation.go`

New endpoints:
- `GET /api/v1/participation/status` - Get current participation status
- `POST /api/v1/participation/select-solo` - Choose solo mode
- `POST /api/v1/participation/transition` - Transition between modes

### 6.2 Update `controller/restapi/v1/team.go`

**Modify** existing endpoints:
- `POST /api/v1/teams/join` - Add `confirm_reset` body param
- `DELETE /api/v1/teams/me` - **New**: Disband team (Captain only)
- `DELETE /api/v1/teams/current/members/{userId}` - **New**: Kick member (Captain only)

**Modify** `controller/restapi/v1/participation.go`:
- `POST /api/v1/participation/transition` - Add `confirm_reset` body param

**Validation**:
- Check `AllowTeamSwitch` config
- Proper error messages including `ErrConfirmationRequired`

### 6.3 Create request/response schemas

**New files:**
- `request/participation.go` - Request schemas for participation endpoints
- `response/participation.go` - Response schemas with status info

---

## Phase 7: Apply Middleware to Routes

### 7.1 Routes requiring team membership

Apply `RequireTeam()` middleware:
- `POST /api/v1/challenges/:id/submit`
- `POST /api/v1/hints/:id/unlock`

### 7.2 Routes requiring email verification

Apply `RequireVerified()` middleware:
- `POST /api/v1/challenges/:id/submit`
- `POST /api/v1/hints/:id/unlock`
- `POST /api/v1/teams` (create)
- `POST /api/v1/teams/join`
- `POST /api/v1/participation/*`

---

## Phase 8: Error Handling

### 8.1 Create new error types in `entity/error/`

New errors:
- `ErrNoTeamSelected` - User hasn't selected participation mode
- `ErrCannotSwitchTeams` - Team switching disabled
- `ErrInvalidTransition` - Invalid state transition
- `ErrSoloModeNotAllowed` - Solo mode disabled for competition
- `ErrTeamModeRequired` - Teams-only competition
- `ErrConfirmationRequired` - Action requires explicit confirmation (data loss warning)
- `ErrConfirmationRequired` - Action requires explicit confirmation (data loss warning)

---

## Phase 9: Testing

### 9.1 Unit Tests

- `usecase/team_test.go` - Test all team transition logic
- `usecase/user_test.go` - Test registration without team creation
- `middleware/require_team_test.go` - Test middleware logic
- `middleware/require_verified_test.go` - Test verification logic

### 9.2 Integration Tests

- `integration-test/team_transition_test.go` - Test state transitions
- `integration-test/participation_test.go` - Test participation flow

### 9.3 E2E Tests

- `e2e-test/team_selection_flow_test.go` - Full user journey
- `e2e-test/solo_to_team_transition_test.go` - Transition scenarios

---

## Phase 10: Configuration

### 10.1 Environment Variables

Add to `.env.example`:
```
VERIFY_EMAILS=false
COMPETITION_MODE=flexible
ALLOW_TEAM_SWITCH=true
MIN_TEAM_SIZE=1
MAX_TEAM_SIZE=10
```

### 10.2 Config Struct

Update `config/config.go`:
- Add verification settings
- Add competition mode settings

---

## Implementation Order

1. ✅ Phase 1: Database migrations
2. ✅ Phase 2: Entity updates
3. ✅ Phase 8: Error types (needed early)
4. ✅ Phase 3: Repository layer
5. ✅ Phase 4: Use case layer
6. ✅ Phase 5: Middleware
7. ✅ Phase 6: API layer
8. ✅ Phase 7: Apply middleware
9. ✅ Phase 10: Configuration
10. ✅ Phase 9: Testing

---

## Breaking Changes

### For Existing Users

- Users with auto-created teams will have `is_auto_created = true`
- These teams will be automatically deleted on first team action
- No data loss - solves and awards are preserved

### For API Clients

- New error codes: `no_team_selected`, `cannot_switch_teams`
- Submit endpoint now requires team membership
- New participation endpoints for team selection

---

## Rollback Plan

If issues arise:
1. Run down migrations in reverse order
2. Restore `UserUseCase.Register()` team creation logic
3. Remove new middleware from routes
4. Remove new API endpoints

---

## Estimated Effort

- Phase 1-2: 30 minutes (migrations + entities)
- Phase 3-4: 2 hours (repository + use cases)
- Phase 5-6: 1.5 hours (middleware + API)
- Phase 7-8: 30 minutes (routing + errors)
- Phase 9: 2 hours (comprehensive testing)
- Phase 10: 15 minutes (configuration)

**Total: ~6.5 hours**

---

## Success Criteria

- ✅ No teams created during registration
- ✅ Users can choose solo/team mode explicitly
- ✅ Solo teams can transition to regular teams
- ✅ Email verification blocks critical actions (when enabled)
- ✅ All existing tests pass
- ✅ New tests cover all transition scenarios
- ✅ Clean database (no orphaned solo teams)
