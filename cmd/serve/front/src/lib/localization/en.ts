import type { Locale, Translations } from '$lib/localization';

const translations = {
	// Authentication
	'auth.signin.title': 'Sign in',
	'auth.signin.description': 'Please fill the form below to access your dashboard.',
	// App
	'app.no_targets': 'No targets found',
	'app.no_targets.description':
		'You need at least one target to deploy your application. Head to the <a href="/targets">create target</a> page to create one.',
	'app.not_found': "Looks like the application you're looking for does not exist. Head back to the",
	'app.not_found.cta': 'homepage',
	'app.blankslate.title': 'Looks like you have no application yet. Start by',
	'app.blankslate.cta': 'creating one!',
	'app.new': 'New application',
	'app.edit': 'Edit application',
	'app.delete': 'Delete application',
	'app.delete.confirm': (name: string) => `Are you sure you want to delete the application ${name}?

This action is IRREVERSIBLE and will DELETE ALL DATA associated with this application on all environments: containers, images, volumes, logs and networks.`,
	'app.delete.failed': 'Deletion failed',
	'app.no_deployments': 'No deployments yet',
	'app.name.help':
		"The application name will determine the subdomain from which deployments will be available. That's why you should <strong>only</strong> use <strong>alphanumeric characters</strong> and a <strong>unique name</strong> accross targets.",
	'app.how': 'How does seelf expose services?',
	'app.how.placeholder.name': '<app-name>',
	'app.how.placeholder.scheme': '<target scheme>://',
	'app.how.placeholder.url': '<target url>',
	'app.how.description':
		'Services with <strong>port mappings defined</strong> will be exposed with the following convention:',
	'app.how.env': 'Environment',
	'app.how.default': 'Default service (first one in alphabetical order)',
	'app.how.others': 'Other exposed services (example: <code>dashboard</code>)',
	'app.how.others.title': 'Other exposed services (example: dashboard)',
	'app.general': 'General settings',
	'app.vcs': 'Version control',
	'app.vcs.enabled': 'Use version control system?',
	'app.vcs.help':
		'If not under version control, you will still be able to manually deploy your application.',
	'app.vcs.token': 'Access token',
	'app.vcs.token.help.instructions':
		'Token used to fetch the provided repository. Generally known as <strong>Personal Access Token</strong>, you can find some instructions for',
	'app.vcs.token.help.leave_empty': ', leave empty if the repository is public.',
	'app.environment.production': 'Production settings',
	'app.environment.staging': 'Staging settings',
	'app.environment.target': 'Deploy target',
	'app.environment.target.changed': 'Target changed',
	'app.environment.target.changed.description': (url: string) =>
		`If you change the target, resources related to this application deployed by seelf on <strong>${url}</strong> will be <strong>REMOVED</strong> and a new deployment on the new target will be queued if possible. If you want to backup something, do it before updating the target.`,
	'app.environment.vars': 'Environment variables',
	'app.environment.vars.service.add': 'Add service variables',
	'app.environment.vars.service.delete': 'Remove service variables',
	'app.environment.vars.service.name': 'Service name',
	'app.environment.vars.service.name.help': (latestServices?: string[]) =>
		'Name of the service as defined in your service file.' +
		(latestServices?.length
			? ` Latest deployed services: ${latestServices?.map((s) => `<code>${s}</code>`).join(', ')}`
			: ''),
	'app.environment.vars.service.env': 'Environment values',
	'app.environment.vars.blankslate': 'No environment variables set.',
	// Target
	'target.not_found': "Looks like the target you're looking for does not exist. Head back to the",
	'target.not_found.cta': 'targets page',
	'target.new': 'New target',
	'target.delete': 'Delete target',
	'target.delete.failed': 'Deletion failed',
	'target.reconfigure': 'Reconfigure',
	'target.reconfigure.failed': 'Reconfiguration failed',
	'target.failed': 'Target configuration has failed',
	'target.ready': 'Target configuration succeeded',
	'target.ready.description': function (at: string) {
		return `Needed infrastructure requested at ${this.datetime(
			at
		)} has been deployed on the target. If something goes wrong, you can click the reconfigure button to try again.`;
	},
	'target.configuring': 'Target configuration in progress',
	'target.configuring.description':
		'Needed infrastructure is being deployed on the target, please wait.',
	'target.blankslate.title': 'Looks like you have no target yet. Start by',
	'target.blankslate.cta': 'creating one!',
	'target.general': 'General settings',
	'target.name.help': 'The name is being used only for display, it can be anything you want.',
	'target.url.help':
		'All applications deployed on this target will be available as a <strong>subdomain</strong> on this root URL (without path). It should be <strong>unique</strong> among targets.',
	'target.provider': 'Provider',
	'target.docker.is_remote': 'Remote docker daemon',
	'target.docker.is_remote.help':
		'Connect to a remote docker daemon using SSH. <strong>Cannot be changed</strong> after creation.',
	'target.docker.host': 'Host',
	'target.docker.host.help':
		'Hostname or IP address of the remote server. <strong>Cannot be changed</strong> after creation.',
	'target.docker.user': 'User',
	'target.docker.port': 'Port',
	'target.docker.private_key': 'Private key',
	'target.docker.private_key.help':
		'Private key to use for SSH connection. <strong>Make sure it includes a newline at the end</strong> or the connection may fail with an <code>invalid format</code> error. You <strong>MUST</strong> add the associated public key to the <code>~/.ssh/authorized_keys</code> on the server side. Passphrase are not supported at the moment.',
	'target.delete.confirm': (name: string) => `Are you sure you want to delete the target ${name}?
	
This action is IRREVERSIBLE and will DELETE ALL DATA on this target, including deployed containers, images, volumes and networks.`,
	'target.delete.confirm_failed_status': (name: string) =>
		`Are you sure you want to delete the target ${name}?
		
Looks like the target is not reachable anymore. If you decide to delete it right now, resources will not be cleaned up.

You may reconsider and try to make the target reachable before deleting it.`,
	'target.reconfigure.confirm':
		'Needed infrastructure will be redeployed on the target. Are you sure?',
	// Jobs
	'jobs.status': 'Status',
	'jobs.resource': 'Message / Resource',
	'jobs.dates': 'Queued at / Not before',
	'jobs.error': 'error',
	'jobs.payload': 'payload',
	'jobs.policy': 'policy',
	'jobs.policy.preserve_group_order': 'Preserve group order on error',
	'jobs.policy.wait_others_resource_id': 'Wait for others jobs to finish on resource',
	'jobs.policy.cancellable': 'Cancellable',
	'jobs.group': 'group',
	'jobs.cancel': 'Cancel job',
	'jobs.cancel.confirm': 'Are you sure you want to cancel this job?',
	// Jobs names
	'deployment.command.cleanup_app': 'Application cleanup',
	'deployment.command.cleanup_target': 'Target cleanup',
	'deployment.command.delete_app': 'Application removal',
	'deployment.command.delete_target': 'Target removal',
	'deployment.command.configure_target': 'Target configuration',
	'deployment.command.deploy': 'Application deployment',
	// Account
	'profile.my': 'my profile',
	'profile.logout': 'log out',
	'profile.password': 'New password',
	'profile.password.help': "Leave empty if you don't want to change your password.",
	'profile.informations': 'Account informations',
	'profile.interface': 'User interface',
	'profile.locale': 'Language',
	'profile.integration': 'Integration',
	'profile.integration.title': 'Integration with CI',
	'profile.integration.description':
		"If you want to trigger a deployment for an application, you'll need this token. You can also click the <strong>Copy cURL command</strong> from the deployment page and use it in your pipeline since it includes the token in the appropriate header.",
	'profile.key': 'API Key',
	'profile.key.help':
		'Pass this token as an <code>Authorization: Bearer</code> header to communicate with the seelf API. <strong>You MUST keep it secret!</strong>',
	// Deployment
	'deployment.new': 'New deployment',
	'deployment.deploy': 'Deploy',
	'deployment.redeploy': 'Redeploy',
	'deployment.redeploy.confirm': (number: number) =>
		`The deployment #${number} will be redeployed. Latest app environment variables will be used. Do you confirm this action?`,
	'deployment.redeploy.failed': 'Redeploy failed',
	'deployment.promote': 'Promote',
	'deployment.promote.confirm': (number: number) =>
		`The deployment #${number} will be promoted to the production environment. Do you confirm this action?`,
	'deployment.promote.failed': 'Promote failed',
	'deployment.blankslate.title': 'No deployment to show. Go ahead and',
	'deployment.blankslate.cta': 'create the first one!',
	'deployment.environment': 'Environment',
	'deployment.payload': 'Payload',
	'deployment.payload.copy_curl': 'Copy cURL command',
	'deployment.payload.kind': 'Kind',
	'deployment.payload.raw': 'compose file',
	'deployment.payload.raw.content': 'Content',
	'deployment.payload.raw.content.help':
		"Content of the service file (compose.yml if you're using Docker Compose for example).",
	'deployment.payload.archive': 'project archive (tar.gz)',
	'deployment.payload.vcs': 'git',
	'deployment.payload.vcs.branch': 'Branch',
	'deployment.payload.vcs.commit': 'Commit',
	'deployment.payload.vcs.commit.help':
		'Optional specific commit to deploy. Leave empty to deploy the latest branch commit.',
	'deployment.logs': 'Deployment logs',
	'deployment.outdated': 'Outdated deployment',
	'deployment.outdated.description':
		"You're viewing an old deployment and exposed URLs may have changed and represent what have been exposed at that time.",
	'deployment.started_at': 'started at',
	'deployment.finished_at': 'finished at',
	'deployment.queued_at': 'queued at',
	'deployment.duration': 'duration',
	'deployment.services': 'deployed services',
	'deployment.branch': 'branch',
	'deployment.commit': 'commit',
	'deployment.error_code': 'error code',
	'deployment.details_tooltip': (number: number) => `View deployment #${number} details and logs`,
	'deployment.not_found':
		"Looks like the deployment you're looking for does not exist! Head back to",
	'deployment.not_found.cta': 'the application page',
	'deployment.waiting_for_logs': 'Waiting for logs...',
	'deployment.target': 'deployed on',
	'deployment.target.deleted': 'target deleted',
	// Menu
	'menu.toggle': 'Toggle menu',
	// Breadcrumb
	'breadcrumb.home': 'Home',
	'breadcrumb.applications': 'Applications',
	'breadcrumb.settings': 'Settings',
	'breadcrumb.application.new': 'New application',
	'breadcrumb.application.settings': (name: string) => `${name} settings`,
	'breadcrumb.deployment.new': 'New deployment',
	'breadcrumb.deployment.name': (number: number) => `Deployment #${number}`,
	'breadcrumb.targets': 'Targets',
	'breadcrumb.target.new': 'New target',
	'breadcrumb.target.settings': (name: string) => `${name} settings`,
	'breadcrumb.jobs': 'Jobs',
	'breadcrumb.profile': 'Profile',
	'breadcrumb.not_found': 'Not found',
	// Footer
	'footer.description': (version: string) =>
		`seelf v${version.substring(0, 16)} — Painless self-hosted deployment platform`,
	'footer.source': 'Source',
	'footer.documentation': 'Docs',
	'footer.donate': '❤️ Donate',
	// Shared
	'panel.hint': 'Click to toggle',
	'datatable.no_data': 'No data to show',
	'datatable.toggle': 'Show / hide details',
	cleanup_requested: 'Marked for deletion',
	'cleanup_requested.description': function (date: DateValue) {
		return `The removal has been requested at ${this.date(date)} and will be processed shortly.`;
	},
	page_n_of_m: (n: number, m: number) => `Page ${n} of ${m}`,
	file: 'File',
	previous: 'Previous',
	next: 'Next',
	and: 'and',
	save: 'Save',
	create: 'Create',
	error: 'Error',
	name: 'Name',
	email: 'Email',
	url: 'Url',
	password: 'Password',
	required: 'Required',
	invalid_email: 'Invalid email',
	unexpected_error: 'An unexpected error occurred.',
	'unexpected_error.description': `<p>Looks like something went wrong. Try refreshing the page.</p><p>If the problem persists, please contact the administrator to investigate.</p>`,
	app_name_already_taken: 'App name is already taken on this target',
	git_branch_not_found: 'Branch not found',
	git_remote_not_reachable: 'Remote not reachable',
	invalid_email_or_password: 'Invalid email or password',
	invalid_app_name: 'Invalid app name',
	url_already_taken: 'Url is already taken',
	config_already_taken: 'A target for this host already exists',
	invalid_host: 'Invalid host',
	invalid_ssh_key: 'Invalid SSH key',
	target_in_use: 'Target is used by at least one application and cannot be deleted.'
} satisfies Translations;

export default {
	code: 'en',
	displayName: 'English',
	translations
} as const satisfies Locale<typeof translations>;
