import type { Environment } from '$lib/resources/deployments';

export type CurlPayload =
	| {
			kind: 'git';
			branch: string;
			hash?: string;
	  }
	| {
			kind: 'archive';
			filename?: string;
	  }
	| {
			kind: 'raw';
			raw: string;
	  };

export type CurlInput = {
	appId: string;
	environment: Environment;
	apiKey: string;
	origin: string;
	payload: CurlPayload;
};

/**
 * Build the cURL command to trigger a deployment.
 */
export function buildCommand({ apiKey, appId, origin, environment, payload }: CurlInput): string {
	let cmd;

	switch (payload.kind) {
		case 'git':
			cmd = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\",\\"git\\":{ \\"branch\\": \\"${
				payload.branch
			}\\"${payload.hash ? `, \\"hash\\": \\"${payload.hash}\\"` : ''} } }" `;
			break;
		case 'archive':
			cmd = `-F environment=${environment} -F archive=@${
				payload.filename ?? '<path_to_a_tar_gz_archive>'
			}`;
			break;
		case 'raw':
			cmd = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${environment}\\", \\"raw\\":\\"${JSON.stringify(
				payload.raw
			)
				.replaceAll('\\"', '"')
				.substring(1)
				.slice(0, -1)}\\"}"`;
			break;
		default:
			return '';
	}

	return `curl -i -X POST -H "Authorization: Bearer ${apiKey}" ${cmd} ${origin}/api/v1/apps/${appId}/deployments`;
}
