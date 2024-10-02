package activity_handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"go.temporal.io/sdk/client"
)

type Handler struct {
	c client.Client
}

func New(c client.Client) (*Handler, error) {
	if c == nil {
		return nil, errors.New("client required and missing")
	}
	return &Handler{c: c}, nil
}

func (h *Handler) VerifyApprover(ctx context.Context, req messages.VerifyApproverRequest) (messages.VerifyApproverResponse, error) {
	if h.c == nil {
		return messages.VerifyApproverResponse{Verified: false}, errors.New("handler misconfigured")
	}
	ev, err := h.c.QueryWorkflow(ctx, req.ApproverID, "", constants.PermissionsGrantedQueryHandlerName)
	if err != nil {
		return messages.VerifyApproverResponse{Verified: false}, err
	}
	m := messages.PermissionsGrantedResponse{}
	err = ev.Get(&m)
	if err != nil {
		return messages.VerifyApproverResponse{Verified: false}, err
	}
	for _, p := range m.Permissions {
		if p == "grant_permissions" {
			return messages.VerifyApproverResponse{Verified: true}, nil
		}
	}
	return messages.VerifyApproverResponse{Verified: false}, nil
}

func (h *Handler) SendNotifications(ctx context.Context, req messages.SendNotificationsRequest) (messages.SendNotificationsResponse, error) {
	fmt.Println("*** SENDING NOTIFICATIONS ***")
	return messages.SendNotificationsResponse{}, nil
}
