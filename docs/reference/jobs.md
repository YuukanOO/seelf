# Jobs

A number of **background tasks** are managed by **seelf**. You can check the jobs page on the dashboard to see them at any time.

For the vast majority of cases, you may never have to look at them as they are processed without issues.

::: info
Some jobs can be delayed if some pre-conditions are not met. For example, when deploying an application, if the [target](/reference/targets) is being configured, the deploy job will be postponed.
:::

## Retrying

Sometimes, things can go wrong.

Jobs in error can be retried manually by pressing the **retry button**.

## Cancellation

Since a target on which you have, in the past, successfully deployed something can be destroyed from your side, **seelf** provides the ability to **dismiss some tasks**.

From the **seelf** perspective, it has effectively deployed something and when, for example, deleting an application, **seelf** will queue a cleanup job which cannot succeed because a target may not be reachable anymore.

For that **particular case**, you can press the **dismiss button** on a job to allow the deletion to proceed without cleaning up resources.
