# Jobs

A number of **background tasks** are managed by **seelf**. You can check the jobs page on the dashboard to see them at any time.

For the vast majority of cases, you may never have to look at them as they are processed without issues.

::: info
By default, **jobs in error** state are retried every **15 seconds**. This is because some errors (such as the `target_configuration_in_progress`) are expected and will delay the job.
:::

## Cancellation

Since a target on which you have, in the past, successfully deployed something can be destroyed from your side, **seelf** provides the ability to **cancel some tasks**.

From the **seelf** perspective, it has effectively deployed something and when, for example, deleting an application, **seelf** will queue a cleanup job which cannot succeed because a target is not reachable anymore.

For that **particular case**, you can press the **cancel button** on a job to allow the deletion to proceed without cleaning up resources.
