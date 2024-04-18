# Frontend

The frontend part of **seelf** is written using [SvelteKit](https://kit.svelte.dev/) for its **low bundle size** and **performances**.

The frontend stuff is located in the `cmd/serve/front` directory and embedded inside the final executable at build time.

## Useful commands

These commands must be executed from the root folder.

- `make serve-front`: Serve the Sveltekit dev server
- `make test`: Run all test suites (front & back), if you only wish to launch the frontend ones, just run `npm test` (in the `cmd/serve/front`) instead
