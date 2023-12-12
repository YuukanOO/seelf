package deploy

// func DeploymentCreatedHandler(b bus.Bus) bus.SignalHandler[domain.DeploymentCreated] {
// 	return func(ctx context.Context, evt domain.DeploymentCreated) error {
// 		_, err := bus.Send(b, ctx, queue.Command{
// 			Message: Command{
// 				AppID:            string(evt.ID.AppID()),
// 				DeploymentNumber: int(evt.ID.DeploymentNumber()),
// 			},
// 			Dedupe: "",
// 		})
// 		return err
// 	}
// }
