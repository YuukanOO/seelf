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
} & CurlPayload;

/**
 * Build the cURL command to trigger a deployment.
 */
export function buildCommand(input: CurlInput): string {
	let payload;

	switch (input.kind) {
		case 'git':
			payload = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${
				input.environment
			}\\",\\"git\\":{ \\"branch\\": \\"${input.branch}\\"${
				input.hash ? `, \\"hash\\": \\"${input.hash}\\"` : ''
			} } }" `;
			break;
		case 'archive':
			payload = `-F environment=${input.environment} -F archive=@${
				input.filename ?? '<path_to_a_tar_gz_archive>'
			}`;
			break;
		case 'raw':
			payload = `-H "Content-Type: application/json" -d "{ \\"environment\\":\\"${
				input.environment
			}\\", \\"raw\\":\\"${JSON.stringify(input.raw)
				.replaceAll('\\"', '"')
				.substring(1)
				.slice(0, -1)}\\"}"`;
			break;
		default:
			return '';
	}

	return `curl -i -X POST -H "Authorization: Bearer ${input.apiKey}" ${payload} ${input.origin}/api/v1/apps/${input.appId}/deployments`;
}
