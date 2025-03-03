import type { Action, ActionReturn } from 'svelte/action';

// FIXME: better typing to enforce that the parameter match the action expected one.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ActionDefinition<Element = HTMLElement, Parameter = any> = [
	Action<Element, Parameter>,
	Parameter
];

/**
 * Enable components to use a svelte action easily.
 * In the future, it may supports multiple actions definition but that's not needed for now.
 */
export function useAction(
	node: HTMLElement,
	action?: ActionDefinition
): void | ActionReturn<ActionDefinition> {
	if (!action) return;

	const result = action[0](node, action[1]);

	if (!result) return;

	return {
		update: ([, parameter]) => {
			result.update?.(parameter);
		},
		destroy: result.destroy
	};
}
