import type { AppTranslations, Locale } from '$lib/localization';

export default {
	code: 'fr',
	displayName: 'Français',
	translations: {
		// Authentication
		'auth.signin.title': 'Connexion',
		'auth.signin.description':
			'Remplissez le formulaire ci-dessous pour accéder au tableau de bord.',
		// App
		'app.not_found':
			"Il semblerait que l'application que vous recherchez n'existe pas. Retournez à la",
		'app.not_found.cta': "page d'accueil",
		'app.blankslate.title': 'Aucune application trouvée, commencez par',
		'app.blankslate.cta': 'en créer une !',
		'app.new': 'Nouvelle application',
		'app.edit': "Modifier l'application",
		'app.delete': "Supprimer l'application",
		'app.delete.confirm': (name: string) => `Voulez-vous vraiment supprimer l'application ${name} ?

Cette action est IRRÉVERSIBLE et supprimera TOUTES LES DONNÉES associées : conteneurs, images, volumes, logs et networks.`,
		'app.delete.failed': 'Erreur de suppression',
		'app.no_deployments': 'Aucun déploiement',
		'app.name.help':
			"Le nom de l'application détermine le sous-domaine utilisé par les déploiements. C'est pourquoi vous devez <strong>uniquement</strong> utiliser des <strong>caractères alphanumériques</strong> et un <strong>nom unique</strong> au sein de l'instance seelf.",
		'app.how': 'Comment les services sont-ils exposés par seelf ?',
		'app.how.placeholder': '<nom-app>',
		'app.how.description':
			'Les services possédant des <strong>mappings de ports</strong> seront exposés selon ces conventions :',
		'app.how.env': 'Environnement',
		'app.how.default': 'Service par défaut (premier par ordre alphabétique)',
		'app.how.others': 'Autres services exposés (exemple: <code>dashboard</code>)',
		'app.how.others.title': 'Autres services exposés (exemple: dashboard)',
		'app.general': 'Paramètres généraux',
		'app.vcs': 'Contrôle de version',
		'app.vcs.enabled': 'Utiliser un contrôle de version ?',
		'app.vcs.help':
			"Si vous n'utilisez pas de contrôle de version, vous pourrez toujours déployer manuellement votre application.",
		'app.vcs.token': "Jeton d'accès",
		'app.vcs.token.help.instructions':
			"Jeton utilisé pour vous authentifier auprès du dépôt. Généralement connu sous le nom de <strong>Jeton d'accès personnel</strong>, vous pouvez trouver des instructions pour",
		'app.vcs.token.help.leave_empty': ', laissez vide si le dépôt est public.',
		'app.environments': "Variables d'environnement",
		'app.environments.help': 'À propos des variables',
		'app.environments.help.description':
			"Les mises à jour des variables d'environnement seront effectives lors du prochain déploiement.",
		'app.environments.service.add': 'Ajouter un service',
		'app.environments.service.delete': 'Supprimer le service',
		'app.environments.service.name': 'Nom du service',
		'app.environments.service.env': "Variables d'environnement",
		'app.environments.blankslate': (name: string) =>
			`Aucune variable pour l'environnement <strong>${name}</strong>.`,
		'app.cleanup_requested': 'Suppression demandée',
		'app.cleanup_requested.description': function (date: DateValue) {
			return `La suppression de l'application a été demandé le ${this.format(
				'date',
				date
			)} et sera traitée sous peu.`;
		},
		// Account
		'profile.my': 'mon profil',
		'profile.logout': 'se déconnecter',
		'profile.password': 'Nouveau mot de passe',
		'profile.password.help': 'Laissez vide si vous ne souhaitez pas changer de mot de passe.',
		'profile.informations': 'Informations du compte',
		'profile.interface': 'Interface utilisateur',
		'profile.locale': 'Langue',
		'profile.integration': 'Intégration',
		'profile.integration.title': 'Intégration Continue',
		'profile.integration.description':
			"Si vous souhaitez déclencher un déploiement pour une application, vous aurez besoin de ce jeton. Vous pouvez également cliquer sur le bouton <strong>Copier la commande cURL</strong> depuis la page de déploiement et l'utiliser dans votre pipeline car il inclut le jeton dans l'en-tête approprié.",
		'profile.key': 'Clé API',
		'profile.key.help':
			"Passez ce jeton dans l'entête <code>Authorization: Bearer</code> pour communiquer avec l'API seelf. <strong>Vous devez le garder secret !</strong>",
		// Deployment
		'deployment.new': 'Nouveau déploiement',
		'deployment.deploy': 'Déployer',
		'deployment.redeploy': 'Redéployer',
		'deployment.redeploy.confirm': (number: number) =>
			`Le déploiement #${number} sera redéployé. La dernière version des variables d'environnement sera utilisée. Confirmez-vous cette action ?`,
		'deployment.redeploy.failed': 'Erreur lors du redéploiement',
		'deployment.promote': 'Promouvoir',
		'deployment.promote.confirm': (number: number) =>
			`Le déploiement #${number} sera promu sur l'environnement de production. Confirmez-vous cette action ?`,
		'deployment.promote.failed': 'Erreur lors de la promotion',
		'deployment.blankslate.title': 'Aucun déploiement trouvé. Commencez par',
		'deployment.blankslate.cta': 'en créer un !',
		'deployment.environment': 'Environnement',
		'deployment.payload': 'Charge utile',
		'deployment.payload.copy_curl': 'Copier la commande cURL',
		'deployment.payload.kind': 'Type',
		'deployment.payload.raw': 'fichier compose',
		'deployment.payload.raw.content': 'Contenu',
		'deployment.payload.raw.content.help':
			'Contenu du fichier de services (compose.yml si vous utilisez Docker Compose par exemple).',
		'deployment.payload.archive': 'archive (tar.gz)',
		'deployment.payload.vcs': 'git',
		'deployment.payload.vcs.branch': 'Branche',
		'deployment.payload.vcs.commit': 'Commit',
		'deployment.payload.vcs.commit.help':
			'Commit spécifique à déployer. Laissez vide pour déployer le dernier commit de la branche.',
		'deployment.logs': 'Logs de déploiement',
		'deployment.outdated': 'Déploiement obsolète',
		'deployment.outdated.description':
			'Vous visualisez un ancien déploiement et les URLs exposées ici représentent ce qui a été exposé au moment du déploiement.',
		'deployment.started_at': 'démarré à',
		'deployment.finished_at': 'terminé à',
		'deployment.queued_at': 'demandé à',
		'deployment.duration': 'durée',
		'deployment.services': 'services exposés',
		'deployment.branch': 'branche',
		'deployment.commit': 'commit',
		'deployment.error_code': 'code erreur',
		'deployment.details_tooltip': (number: number) =>
			`Voir les détails et logs du déploiement #${number}`,
		'deployment.not_found':
			"Il semblerait que le déploiement que vous recherchez n'existe pas. Retournez à",
		'deployment.not_found.cta': "la page d'application",
		// Breadcrumb
		'breadcrumb.applications': 'Applications',
		'breadcrumb.settings': 'Paramètres',
		'breadcrumb.application.new': 'Nouvelle application',
		'breadcrumb.application.settings': (name: string) => `Paramètres de ${name}`,
		'breadcrumb.deployment.new': 'Nouveau déploiement',
		'breadcrumb.deployment.name': (number: number) => `Déploiement #${number}`,
		'breadcrumb.profile': 'Profil',
		'breadcrumb.not_found': 'Ressource introuvable',
		// Footer
		'footer.description': 'seelf — Plateforme de déploiement auto-hébergée',
		'footer.source': 'Source',
		'footer.documentation': 'Docs',
		'footer.donate': '❤️ Donner',
		// Shared
		page_n_of_m: (n: number, m: number) => `Page ${n} de ${m}`,
		file: 'Fichier',
		previous: 'Précédent',
		next: 'Suivant',
		and: 'et',
		save: 'Enregistrer',
		create: 'Créer',
		error: 'Erreur',
		name: 'Nom',
		email: 'Email',
		url: 'Url',
		password: 'Mot de passe',
		required: 'Requis',
		invalid_email: 'Email invalide',
		unexpected_error: 'Une erreur imprévue est survenue.',
		'unexpected_error.description':
			"<p>Quelque chose s'est mal passé. Tentez de rafraichir la page.</p><p>Si le problème persiste, veuillez contacter l'administrateur.</p>",
		app_name_already_taken: "Nom d'application déjà utilisé",
		git_branch_not_found: 'Branche non trouvée',
		invalid_email_or_password: 'Email ou mot de passe invalide'
	}
} as const satisfies Locale<AppTranslations>;
