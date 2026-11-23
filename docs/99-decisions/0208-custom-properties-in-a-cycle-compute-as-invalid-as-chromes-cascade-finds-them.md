# 0208. Custom properties in a cycle compute as invalid, as Chrome's cascade finds them

- Status: Accepted
- Date: 2026-11-23
- Area: api
- Owner: engine

## Context

A registered custom property of a length syntax holding `em` computes against the element's font-size, so `font-size: var(--x)` with `--x: 2em` makes a cycle, as `lh` in one read by `line-height` does. ui resolved the custom properties first with the parent's font, then again with the element's, and let font-size take what came out; in a `var()` cycle only the property found to close it was invalid, and the others computed from it.

## Decision

- The cascade keeps a stack of what it is resolving. font-size, then line-height, resolve the custom properties they read before the rest, as Chrome applies them first; a registered property whose value, once substituted, holds a unit of the font (`em`, `ex`, `ch`, `cap`, `ic`, `lh`, and on the root the root's units) closes a cycle with font-size when font-size is on the stack, and one holding `lh` (`rlh` on the root) with line-height.
- Every property on the stack from where the cycle closes is in it, custom properties between them included, for `var()` cycles too.
- A longhand in a cycle is unset. A registered custom property in one computes as invalid: inherited where it inherits, else its initial value; an unregistered one, and one whose `var()` has nothing and no fallback, is the guaranteed-invalid value, no longer its parent's.
- A registered property reading `lh` outside a cycle takes the element's own line-height, worked out after its font-size.

## Alternatives

- A dependency graph of every custom property and longhand, cut by strongly connected components as the spec words it: finds the same cycles, but costs every element a graph Chrome's stack gets for free, and Chrome is what pages are tested against.
- Keeping the two passes: the font-size reading a cyclic property came out as the property's second value, where Chrome inherits the parent's.

## Consequences

Font and line-height unit cycles, `var()` cycles and what reads a property in one compute as Chrome's (`TestUnitCyclesComputeAsInChrome`). Elements without registered properties pay nothing; those with own custom properties keep a short stack.
