package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	c  client.Client
	ns string
}

type Option func(*Handler)

func New(c client.Client, namespace string, opts ...Option) (*Handler, error) {
	h := &Handler{
		c:  c,
		ns: namespace,
	}
	if h.c == nil {
		return nil, errors.New("temporal client required & missing")
	}
	for _, o := range opts {
		o(h)
	}
	return h, nil
}

func (h Handler) GETApprovePermission(gc *gin.Context) {
	gc.HTML(http.StatusOK, "approve_permission.html", nil)
}

func (h Handler) GETCreateUser(gc *gin.Context) {
	gc.HTML(http.StatusOK, "create_user.html", nil)
}

func (h Handler) GETRequestPermission(gc *gin.Context) {
	gc.HTML(http.StatusOK, "request_permission.html", nil)
}

func (h Handler) GETUser(gc *gin.Context) {
	if gc.Query("id") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "id required and missing")
		return
	}
	ev, err := h.c.QueryWorkflow(gc.Request.Context(), gc.Query("id"), "",
		constants.UserDetailsQueryHandlerName)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ud := messages.UserDetailsResponse{}
	err = ev.Get(&ud)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	gc.HTML(http.StatusOK, "user.html", messages.GETUserResponse{
		AwaitingApproval:   ud.AwaitingApproval,
		DeletionRequested:  ud.DeletionRequested,
		DeletionUndoWindow: ud.DeletionScheduledFor.Sub(time.Now().UTC()).String(),
		Permissions:        ud.Permissions,
		Username:           gc.Query("id"),
	})
}

func (h Handler) GETUsers(gc *gin.Context) {
	// In a production app we ought to paginate through the executions. Since this is a demo we can assume that the
	// number of open executions will fit in a single request.
	listWorkflowReq := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: h.ns,
		Query:     "`ExecutionStatus`=\"Running\"",
	}
	if gc.Query("permission") != "" {
		queryString := "`ExecutionStatus`=\"Running\" AND `permissions`=\"{NAME}\""
		queryString = strings.Replace(queryString, "{NAME}", gc.Query("permission"), 1)
		listWorkflowReq.Query = queryString
	}
	listResponse, err := h.c.ListWorkflow(gc.Request.Context(), listWorkflowReq)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	type User struct {
		Username string
	}
	users := make([]User, 0)
	for _, e := range listResponse.GetExecutions() {
		users = append(users, User{Username: e.Execution.WorkflowId})
	}
	gc.HTML(http.StatusOK, "users.html", users)
}

func (h Handler) POSTApprovePermission(gc *gin.Context) {
	if gc.PostForm("requester_username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "requester_username required and missing")
		return
	}
	if gc.PostForm("approver_username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "approver_username required and missing")
		return
	}
	if gc.PostForm("permission_type") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "permission_type required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("requester_username"),
		UpdateName: constants.ApproveUserPermissionUpdateHandlerName,
		Args: []interface{}{
			&messages.ApproveUserPermissionRequest{
				ApproverID: gc.PostForm("approver_username"),
				Permission: gc.PostForm("permission_type"),
			},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := messages.ApproveUserPermissionResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("requester_username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTCreateUser(gc *gin.Context) {
	if gc.Request.FormValue("username") == "" {
		gc.String(http.StatusBadRequest, "username required and missing")
		return
	}
	opts := client.StartWorkflowOptions{
		ID:        gc.Request.FormValue("username"),
		TaskQueue: constants.EntityTaskQueueName,
	}
	workflowInput := messages.UserAccountOrchestrationInput{
		Permissions:      make([]string, 0),
		AwaitingApproval: make([]string, 0),
	}

	run, err := h.c.ExecuteWorkflow(gc.Request.Context(), opts, "Orchestration", workflowInput)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if gc.Request.FormValue("make_user_approver") == "on" {
		updateOptions := client.UpdateWorkflowOptions{
			WorkflowID: run.GetID(),
			UpdateName: constants.CreateUserAccountUpdateHandlerName,
			Args: []interface{}{
				&messages.CreateUserAccountRequest{
					Permissions: []string{constants.PermissionTypeGrantPermissions},
				},
			},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		}
		updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
		if err != nil {
			_ = gc.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		updateResponse := &messages.CreateUserAccountResponse{}
		err = updateHandle.Get(gc.Request.Context(), &updateResponse)
		if err != nil {
			_ = gc.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	gc.Redirect(http.StatusSeeOther, "/users")
}

func (h Handler) POSTDeleteUser(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.DeleteUserAccountUpdateHandlerName,
		Args: []interface{}{
			&messages.DeleteUserAccountRequest{},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.DeleteUserAccountResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTRequestPermission(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	if gc.PostForm("permission_type") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "permission_type required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.AddUserPermissionUpdateHandlerName,
		Args: []interface{}{
			&messages.AddUserPermissionRequest{
				Permission: gc.PostForm("permission_type"),
			},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.AddUserPermissionResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTUndoDeleteUser(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.UndoDeleteUserAccountUpdateHandlerName,
		Args: []interface{}{
			&messages.UndoDeleteUserAccountRequest{},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.UndoDeleteUserAccountResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}
