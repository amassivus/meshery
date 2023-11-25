package machines

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/layer5io/meshery/server/models"
	"github.com/layer5io/meshery/server/models/connections"
	"github.com/layer5io/meshkit/models/events"
	"github.com/layer5io/meshkit/utils"
	"github.com/sirupsen/logrus"
)

// Action to be executed in a given state.
type Action interface {

	// Used as guards/prerequisites checks and actions to be performed when the machine enters a given state.
	ExecuteOnEntry(context context.Context, machinectx interface{}, data interface{}) (EventType, *events.Event, error)

	Execute(context context.Context, machinectx interface{}, data interface{}) (EventType, *events.Event, error)

	// Used for cleanup actions to perform when the machine exits a given state
	ExecuteOnExit(context context.Context, machinectx interface{}, data interface{}) (EventType, *events.Event, error)
}

type DefaultConnectAction struct{}

func (da *DefaultConnectAction) ExecuteOnEntry(ctx context.Context, machineCtx interface{}, data interface{}) (EventType, *events.Event, error) {
	return NoOp, nil, nil
}

func (da *DefaultConnectAction) Execute(ctx context.Context, machineCtx interface{}, data interface{}) (EventType, *events.Event, error) {
	sysID := uuid.Nil
	userUUID := uuid.Nil
	token, _ := ctx.Value(models.TokenCtxKey).(string)

	eventBuilder := events.NewEvent().ActedUpon(userUUID).WithCategory("connection").WithAction("update").FromSystem(sysID).FromUser(userUUID).WithDescription("Failed to interact with the connection.")

	provider, _ := ctx.Value(models.ProviderCtxKey).(models.Provider)

	payload, err := utils.Cast[Payload](data)
	if err != nil {
		return NoOp, eventBuilder.WithSeverity(events.Error).WithMetadata(map[string]interface{}{"error": err}).Build(), err
	}

	credential, err := provider.SaveUserCredential(token, &models.Credential{
		Name:   payload.Credential.Name,
		UserID: payload.Credential.UserID,
		Type:   payload.Credential.Type,
		Secret: payload.Credential.Secret,
	})

	if err != nil {
		_err := models.ErrPersistCredential(err)
		return NoOp, eventBuilder.WithDescription(fmt.Sprintf("Unable to persist credential information for the connection %s", payload.Credential.Name)).
			WithSeverity(events.Error).WithMetadata(map[string]interface{}{"error": err}).Build(), _err
	}

	connection, err := provider.SaveConnection(&models.ConnectionPayload{
		Kind:         payload.Connection.Kind,
		Type:         payload.Connection.Type,
		SubType:      payload.Connection.SubType,
		Status:       connections.CONNECTED,
		Name:         payload.Connection.Name,
		MetaData:     payload.Connection.Metadata,
		CredentialID: credential.ID,
	}, token, false)
	if err != nil {
		_err := models.ErrPersistConnection(err)
		return NoOp, eventBuilder.WithDescription(fmt.Sprintf("Unable to perisit the \"%s\" connection details", payload.Connection.Name)).WithMetadata(map[string]interface{}{"error": _err}).Build(), _err
	}
	logrus.Debug(connection, "grafana connected connection")
	return NoOp, nil, nil
}

func (da *DefaultConnectAction) ExecuteOnExit(ctx context.Context, machineCtx interface{}, data interface{}) (EventType, *events.Event, error) {
	return NoOp, nil, nil
}
