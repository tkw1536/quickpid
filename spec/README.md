# PID Resolver Specification

> [!WARNING]
> This README is still a work in progress and incomplete.
> See [the main README](../README.md) for a generic introduce to what PID is and why it is needed.

![Architectural Sketch Of The PID system](pid_arch.svg "The PID System Architecture")

We propose that a PID system consist out of the following components, also seen in the sketch above:

- An internal __PID Resolver API__ and associated database backend.
  It is the central system that handles issuing and storing PIDs and associated metadata.
- A public __Read-Only Frontend__ that can display each PID and respond to clients with a http redirect response for specific PIDs.  
- A __Customer-Facing API__, which improves upon the usability of the internal API, as well as handle authentication and authorization.
- Further __Internal Clients__, which connect directly to the API.

This folder only provides a technical documentation and specification for the PID Resolver API.
These were written up by me (Tom Wiesing).

The system as a whole, and the PID Resolver API specifically, were designed collaboratively with input, feedback, and discussion from (in alphabetical order):
<!-- spellchecker:words Dominik Schmid Amann Walther -->
- Ann-Christine Planck
- Dominik Schmid
- Kai Amann
- Marcus Walther
- Mona Dietrich

Content in this directory is available under either of the following two licenses:

- [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/), see [the LICENSE file](./LICENSE).
- [GNU Affero General Public License 3.0](https://www.gnu.org/licenses/agpl-3.0.en.html) (to match the rest of this repository) see [Top-Level LICENSE file](../LICENSE).

## API Overview

The API specification takes the shape of an [OpenAPI 3.0.0](https://spec.openapis.org/oas/v3.0.0.html) specification in [`openapi.yaml`](./openapi.yaml).
Information in this README is non-normative, and provided for convenience only.

## Testcases

Testcases can be found in the [`tests`](`./tests/`) directory.
