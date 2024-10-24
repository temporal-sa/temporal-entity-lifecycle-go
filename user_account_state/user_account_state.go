package user_account_state

import (
	"errors"
	"fmt"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type Option func(*UserAccountState)

type UserAccountState struct {
	awaitingApproval     []string
	ctx                  workflow.Context
	created              bool
	deleted              bool
	deletionRequested    bool
	deletionRequestedAt  time.Time
	deletionScheduledFor time.Time
	logger               log.Logger
	permissionsGranted   []string
}

func New(ctx workflow.Context, opts ...Option) (*UserAccountState, error) {
	state := &UserAccountState{
		ctx: ctx,
	}
	for _, o := range opts {
		o(state)
	}
	if state.ctx == nil {
		return nil, errors.New("context required and missing")
	}
	state.logger = workflow.GetLogger(state.ctx)
	if len(state.permissionsGranted) > 0 {
		err := state.refreshSearchAttributes()
		if err != nil {
			return nil, err
		}
	}
	if !state.deletionRequestedAt.IsZero() {
		state.RequestDeletion(messages.DeleteUserAccountRequest{})
	}
	return state, nil
}

func WithSnapshot(input messages.UserAccountOrchestrationInput) Option {
	return func(state *UserAccountState) {
		state.awaitingApproval = input.AwaitingApproval
		state.permissionsGranted = input.Permissions
		state.deletionRequestedAt = input.DeletionRequestedAt
	}
}

func (state *UserAccountState) refreshSearchAttributes() error {
	var errs error
	permissionsKey := temporal.NewSearchAttributeKeyKeywordList(constants.PermissionsSearchAttributeKey)
	err := workflow.UpsertTypedSearchAttributes(state.ctx, permissionsKey.ValueSet(state.permissionsGranted))
	if err != nil {
		errs = errors.Join(errs, err)
	}
	approvalsKey := temporal.NewSearchAttributeKeyKeywordList(constants.AwaitingApprovalSearchAttributeKey)
	err = workflow.UpsertTypedSearchAttributes(state.ctx, approvalsKey.ValueSet(state.awaitingApproval))
	if err != nil {
		errs = errors.Join(errs, err)
	}
	return errs
}

func (state *UserAccountState) userHasPermissionPendingApproval(permission string) bool {
	for _, pending := range state.awaitingApproval {
		if permission == pending {
			return true
		}
	}
	return false
}

func (state *UserAccountState) AwaitingApproval() messages.AwaitingApprovalResponse {
	return messages.AwaitingApprovalResponse{Permissions: state.awaitingApproval}
}

func (state *UserAccountState) CreateUser(req messages.CreateUserAccountRequest) error {
	if state.deleted || state.deletionRequested {
		return errors.New("user deleted")
	}
	state.permissionsGranted = append(state.permissionsGranted, req.Permissions...)
	state.created = true
	return state.refreshSearchAttributes()
}

func (state *UserAccountState) Deleted() bool {
	return state.deleted
}

func (state *UserAccountState) DeletionRequestedAt() time.Time {
	return state.deletionRequestedAt
}

func (state *UserAccountState) Permissions() messages.PermissionsGrantedResponse {
	return messages.PermissionsGrantedResponse{Permissions: state.permissionsGranted}
}

func (state *UserAccountState) RequestAddPermission(req messages.AddUserPermissionRequest) error {
	if state.deleted || state.deletionRequested {
		return errors.New("user deleted")
	}
	state.awaitingApproval = append(state.awaitingApproval, req.Permission)
	return state.refreshSearchAttributes()
}

func (state *UserAccountState) RequestApprovePermission(ctx workflow.Context, req messages.ApproveUserPermissionRequest) error {
	if state.deleted || state.deletionRequested {
		return errors.New("user deleted")
	}
	if state.userHasPermissionPendingApproval(req.Permission) {
		resp := messages.VerifyApproverResponse{}
		actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 1 * time.Minute,
		})
		err := workflow.ExecuteActivity(actCtx, "VerifyApprover", &messages.VerifyApproverRequest{
			ApproverID: req.ApproverID,
			Permission: req.Permission,
		}).Get(actCtx, &resp)
		if err != nil {
			return err
		}
		// **** uncomment for versioning & replay testing demo
		//v := workflow.GetVersion(ctx, "add_send_notifications_activity", workflow.DefaultVersion, 0)
		//if v != workflow.DefaultVersion {
		//	err = workflow.ExecuteActivity(actCtx, "SendNotifications", &messages.SendNotificationsRequest{
		//		ApproverID:     req.ApproverID,
		//		PermissionType: req.Permission,
		//		RequesterID:    workflow.GetInfo(state.ctx).WorkflowExecution.ID,
		//	}).Get(actCtx, &resp)
		//}
		if resp.Verified {
			state.permissionsGranted = append(state.permissionsGranted, req.Permission)
			awaitingApproval := make([]string, 0)
			for _, a := range state.awaitingApproval {
				if a != req.Permission {
					awaitingApproval = append(awaitingApproval, a)
				}
			}
			state.awaitingApproval = awaitingApproval
			err := state.refreshSearchAttributes()
			if err != nil {
				state.logger.Error("unable to refresh search attributes", err)
			}
		} else {
			return errors.New(fmt.Sprintf("%s cannot grant permission %s", req.ApproverID, req.Permission))
		}
	} else {
		return errors.New("permission not found")
	}
	return nil
}

func (state *UserAccountState) RequestDeletion(_ messages.DeleteUserAccountRequest) {
	undoDeletionWindow := time.Second * 60
	state.deletionRequested = true
	workflow.Go(state.ctx, func(inner workflow.Context) {
		state.deletionRequestedAt = workflow.Now(inner)
		state.deletionScheduledFor = state.deletionRequestedAt.Add(undoDeletionWindow)
		ok, err := workflow.AwaitWithTimeout(inner, undoDeletionWindow, func() bool {
			// AwaitWithTimeout uses a durable timer under the hood
			return !state.deletionRequested
		})
		if err != nil {
			state.logger.Info("timer cancelled", err)
		}
		if !ok {
			state.deleted = true
		}
	})
}

func (state *UserAccountState) RequestUndoDeletion(_ messages.UndoDeleteUserAccountRequest) error {
	if state.deleted {
		return errors.New("already deleted")
	}
	state.deletionRequested = false
	return nil
}

func (state *UserAccountState) UserDetails() messages.UserDetailsResponse {
	resp := messages.UserDetailsResponse{
		AwaitingApproval:     state.AwaitingApproval(),
		DeletionRequested:    state.deletionRequested,
		DeletionRequestedAt:  state.deletionRequestedAt,
		DeletionScheduledFor: state.deletionScheduledFor,
		Permissions:          state.Permissions(),
	}
	return resp
}
