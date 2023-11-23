import type { Locale, Translations } from '$lib/localization';

const translations = {
	// Authentication
	'auth.signin.title': 'Sign in',
	'auth.signin.description': 'Please fill the form below to access your dashboard.',
	// App
	'app.not_found': "Looks like the application you're looking for does not exist. Head back to the",
	'app.not_found.cta': 'homepage',
	'app.blankslate.title': 'Looks like you have no application yet. Start by',
	'app.blankslate.cta': 'creating one!',
	'app.new': 'New application',
	'app.edit': 'Edit application',
	'app.delete': 'Delete application',
	'app.delete.confirm': (name: string) => `Are you sure you want to delete the application ${name}?

This action is IRREVERSIBLE and will DELETE ALL DATA associated with this application: containers, images, volumes, logs and networks.`,
	'app.delete.failed': 'Deletion failed',
	'app.no_deployments': 'No deployments yet',
	'app.name.help':
		"The application name will determine the subdomain from which deployments will be available. That's why you should <strong>only</strong> use <strong>alphanumeric characters</strong> and a <strong>unique name</strong> accross seelf.",
	'app.how': 'How does seelf expose services?',
	'app.how.placeholder': '<app-name>',
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
	'app.environments': 'Environment variables',
	'app.environments.help': 'Note about variables',
	'app.environments.help.description':
		'Updates to your application environment variables will be effective on the next deployment.',
	'app.environments.service.add': 'Add service variables',
	'app.environments.service.delete': 'Remove service variables',
	'app.environments.service.name': 'Service name',
	'app.environments.service.env': 'Environment values',
	'app.environments.blankslate': (name: string) =>
		`No environment variables set for environment <strong>${name}</strong>.`,
	'app.cleanup_requested': 'Marked for deletion',
	'app.cleanup_requested.description': function (date: DateValue) {
		return `The application removal has been requested at ${this.format(
			'date',
			date
		)} and will be processed shortly.`;
	},
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
	// Breadcrumb
	'breadcrumb.applications': 'Applications',
	'breadcrumb.settings': 'Settings',
	'breadcrumb.application.new': 'New application',
	'breadcrumb.application.settings': (name: string) => `${name} settings`,
	'breadcrumb.deployment.new': 'New deployment',
	'breadcrumb.deployment.name': (number: number) => `Deployment #${number}`,
	'breadcrumb.profile': 'Profile',
	'breadcrumb.not_found': 'Not found',
	// Footer
	'footer.description': 'seelf — Painless self-hosted deployment platform',
	'footer.source': 'Source',
	'footer.documentation': 'Docs',
	'footer.donate': '❤️ Donate',
	// Shared
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
	unexpected_error: 'An unexpected error occured.',
	'unexpected_error.description': `<p>Looks like something went wrong. Try refreshing the page.</p><p>If the problem persists, please contact the administrator to investigate.</p>`,
	app_name_already_taken: 'App name is already taken',
	git_branch_not_found: 'Branch not found',
	invalid_email_or_password: 'Invalid email or password'
} satisfies Translations;

export default {
	code: 'en',
	displayName: 'English',
	translations
} as const satisfies Locale<typeof translations>;
