import type { ActionReturn } from 'svelte/action';

export type AriaControlsParameter = {
	expanded: boolean;
	controls: string;
};

/**
 * Action to controls aria-expanded and aria-controls attributes.
 */
export function controls(
	node: HTMLElement,
	parameters: AriaControlsParameter
): ActionReturn<AriaControlsParameter> {
	function updateAriaAttributes({ expanded, controls }: AriaControlsParameter) {
		if (expanded) {
			node.setAttribute('aria-expanded', 'true');
			node.setAttribute('aria-controls', controls);
		} else {
			node.setAttribute('aria-expanded', 'false');
			node.removeAttribute('aria-controls');
		}
	}

	updateAriaAttributes(parameters);

	return {
		update: updateAriaAttributes
	};
}
