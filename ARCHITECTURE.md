# Architecture overview

This document explains some decisions behind the codebase that you'll see there.

## Stack

The backend code is written in **Go** for its simplicity and small footprint. The frontend use **SvelteKit** for its small bundles to keep seelf as tiny as possible.

## Packages overview

- `cmd/`: contains application commands such as the http backend server
- `internal/`: contains internal package representing the core features of this application organized by bounded contexts and `app`, `domain` and `infra` folders (see [The Domain](#the-domain))
- `pkg/`: contains reusable stuff

## The Domain

I'm a big fan of [Domain Driven Design](https://en.wikipedia.org/wiki/Domain-driven_design) and as such, this codebase is heavily influenced by that with minor tweaks to make it more Go friendly.

The `internal/` follows a classic DDD structure with:

- `app`: commands and queries to orchestrate the domain logic
- `domain`: core stuff, entities and values objects, as pure as possible to be easily testable
- `infra`: implementation of domain specific interfaces for the current context

In Go, it's common to see entities as structs with every field exposed. In this project, I have decided to try something else to prevent unwanted mutations from happening.

Domain entities **encapsulates** a lot of rules and should always **be in a valid state** at all cost. Entities only expose needed fields and methods to mutate them and keep their invariants true. When they mutate, they raise **events** (stored directly inside them) representing what have been changed on an entity. Events enable the system to _react_ to entities mutation (See [Persistence](#persistence)).

Value objects regroup multiple properties that operate together into one [immutable](#immutability) struct acting as a whole and keeping their own invariants true.

The **rule of thumbs** in this project **regarding struct creation** is to **always pass by a constructor function** if any. I could have enforced the valid creation by hiding struct behind an interface but that's a lot of additional complexity.

## Immutability

For entities, mutation is allowed because that's the most reasonnable way to think about them, that's why mutating methods operates on a pointer receiver whereas Value Objects are immutable and always manipulate struct by values.

## Persistence

Only entities could be persisted. Since they raise events representing what have changed after a mutation, those events make their way to **entity stores** (defined as interfaces in `domain` packages and implemented in `infra` packages) where they are translated to SQL queries.

This make it easy to execute surgical SQL updates as needed.

Every entities should be read from the persistent store as a whole (= it should be populated with all their fields set). In this project, every entity expose a method which takes a `storage.Scanner` and returns an entity of the given type. This method, since it needs access to unexposed fields, is defined next to the public _constructor_ of an entity in the `domain` sub-package.

Some value objects implements the `Scanner`, `Valuer`, `Marshaler` and `Unmarshaler` interfaces when they must be persisted in a single column. I may eventualy found another cleaner way to do this but this is sufficient for now.

Retrieving related data is a complex story using the `Scanner` approach and unexported fields constraint. I made it work by enabling a custom `Scanner` to be defined and scan additional related entities. Those custom ones are implemented in the infrastructure layer and retrieve additional relations by using something inspired by graphql dataloaders to prevent N+1 queries.

## Commands and DataGateway

The domain is never accessed directly by client applications (here, a REST API). That's why there's `app` packages in every domain represented by an `internal/` sub-package.

`app` packages expose:

- `command/`: mutate the system based on given inputs, only take and returns primitive types. Commands translate application usecases to domain actions.
- `query/`: read stuff from the database using `Gateway` interfaces. Taken and returned types are defined in the package itself since the representation differ from the mutating stuff. Here, every field is public since it will be serialized afterwards and represents only UI needs.

## Optionality

To make things more explicit, optional values are not represented using a pointer but a specific `monad.Maybe[T]` type instead. This type implements some common interfaces such as `Scanner`, `Valuer`, `Marshaler` and `Unmarshaler` to enable persistence and JSON serialization.
