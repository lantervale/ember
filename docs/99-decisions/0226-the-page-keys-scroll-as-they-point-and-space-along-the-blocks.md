# 0226. The page keys scroll as they point, and Space along the blocks

- Status: Accepted
- Date: 2026-11-24
- Area: api
- Owner: engine

## Context

Decision 0221 had Page Up, Page Down, Home and End move a scroller along its blocks off macOS, an assumption no Chrome had checked. Chrome on Linux pages and ends a `vertical-rl` page physically down, where there is nothing to scroll, as it pages on macOS.

## Decision

- Page Up and Page Down scroll up and down the page on every platform, whatever its writing mode.
- Home and End do too, but on macOS, where Chrome runs them as the editing commands that scroll to the document's start and end along its blocks.
- Space, back with Shift, pages along the blocks everywhere. The arrows and the wheel scroll as they point.

## Alternatives

- Keeping every key along the blocks off macOS: what a reader of a vertical page might expect, but no Chrome does it.

## Consequences

A vertical root scrolls by keys as Chrome on each platform; Space is the one key that pages it along its blocks on Linux and Windows.
