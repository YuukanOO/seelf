package deploy

// func OnDeploymentsFailedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.DeploymentsFailed] {
// 	return func(ctx context.Context, evt domain.DeploymentsFailed) error {
// 		return scheduler.Cancel(ctx, collections.Map(evt.Deployments, func(d domain.DeploymentID) bus.JobKey {
// 			return jobKey(string(d.AppID()), int(d.DeploymentNumber()))
// 		}))
// 	}
// }
