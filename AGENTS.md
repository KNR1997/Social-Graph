# Relationship Graph - AGENTS.md

## Project Overview
A personal relationship management system that models relationships as a connected graph. Source of truth: `README.md`.

## Current State
- **Status**: Early development / concept (see README: "🚧 Early Development / Concept")
- **No source code yet** - this is a planning/design phase repository
- **Primary documentation**: `README.md` (849 lines, covers domain model, architecture, features, roadmap)

## What an Agent Should Know
- The README defines the full domain: Persons, Relationships, Interactions, Relationship Health, Reminders, Reconnection Suggestions
- Planned tech stack (from README): Go/Python backend, React/Next.js frontend, PostgreSQL, Docker
- Architecture direction: DDD, Clean Architecture, REST APIs, Event-Driven Architecture, Graph database queries
- Roadmap has 5 phases from Foundation through Automation
- No build system, test framework, or linting configured yet

## High-Signal Conventions (to be established)
- None yet - project has no code. When source is added, follow the architecture direction from README.md §"Architecture" section.
- When code is added, the domain model entities are: Person, Relationship, Interaction, RelationshipType, Reminder, ImportantEvent
- Value objects: RelationshipStrength, RelationshipHealth, InteractionFrequency, ContactMethod

## Commands (none yet - add as project evolves)
No developer commands exist. When the project gains a build/test system, document exact commands here.

## What to Avoid
- Do NOT assume any source code, package manager, or test framework exists
- Do NOT add fluff or speculative claims - only verified facts from README.md