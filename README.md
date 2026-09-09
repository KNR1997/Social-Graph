# Relationship Graph

> A personal relationship management system that helps you understand, maintain, and strengthen the relationships in your life.

## Overview

Relationships don't disappear overnight. They usually fade gradually when people stop interacting, sharing experiences, or making an effort to stay connected.

**Relationship Graph** is a personal relationship management application designed to help people understand their social network and maintain meaningful relationships over time.

Instead of treating people as a simple contact list, the application models relationships as a **connected graph**.

For example:

```text
John
 │
 └── Jason
      │
      └── Paul
```

John may initially know Jason from college. Through Jason, John meets Jason's older brother Paul at a party. John can then decide to establish and maintain a relationship with Paul independently.

Over time, John's network might become:

```text
                         ┌── Paul
                         │
John ───── Jason ────────┼── Sarah
                         │
                         └── David
                              │
                              └── Michael
```

The application helps John understand these connections, remember important interactions, identify relationships that may need attention, and determine meaningful ways to reconnect.

---

## Problem

Modern contact applications mostly answer:

> "Who do I know?"

They don't answer:

* How do I know this person?
* When did we last interact?
* How important is this relationship to me?
* How frequently do we normally interact?
* What did we talk about last time?
* Who introduced us?
* Which relationships are becoming inactive?
* How should I reconnect with someone?
* What other connections exist through my current relationships?

As our social and professional networks grow, remembering and maintaining these relationships becomes increasingly difficult.

**Relationship Graph aims to solve this problem.**

---

## Core Idea

The fundamental idea is:

> **Relationships require maintenance, and meaningful relationships can be represented as a connected network.**

The system maintains information about:

```text
Person
   │
   ├── Relationship
   │      ├── Type
   │      ├── Strength
   │      ├── Importance
   │      ├── Started At
   │      └── Introduced By
   │
   └── Interactions
          ├── Message
          ├── Call
          ├── Meeting
          └── Event
```

The system can then use this information to help users maintain their relationships intentionally.

---

# Features

## 👤 People

Maintain profiles for people in your network.

Possible information includes:

* Name
* Contact information
* Occupation
* Organization
* Location
* Interests
* Notes
* Important dates

---

## 🔗 Relationships

Create relationships between people.

Examples:

* Friend
* Family
* College
* School
* Colleague
* Mentor
* Mentee
* Professional connection
* Neighbor
* Acquaintance

A relationship can also record **how the relationship was established**.

Example:

```text
John → Paul

Relationship: Friend
Met through: Jason
First interaction: 2026-01-12
Importance: High
Preferred interaction frequency: 30 days
```

---

## 🧑‍🤝‍🧑 Connection Graph

Relationships form a graph rather than a flat contact list.

Example:

```text
John
 │
 ├── Jason
 │    │
 │    ├── Paul
 │    └── Sarah
 │
 └── Mike
      │
      └── David
```

This allows users to understand:

* Direct relationships
* Mutual connections
* Extended connections
* How two people are connected
* How a person entered their network

---

## 📝 Interaction History

Record meaningful interactions with people.

Examples:

```text
September 5
Coffee with Paul
Discussed his new startup.

August 12
Called Jason
Talked about university.

July 20
Met Sarah at a conference.
```

Supported interaction types could include:

* Message
* Phone call
* Video call
* Meeting
* Coffee
* Dinner
* Event
* Gift
* Other

---

## ❤️ Relationship Health

The system can estimate the current health of a relationship based on multiple factors.

For example:

```text
Paul

Relationship Health
████████░░  78%

Last interaction: 18 days ago
Typical frequency: 30 days
Importance: High
Recent interactions: 4
```

The health score should not depend only on time.

A relationship can remain strong even when people don't communicate frequently.

Potential factors include:

```text
Relationship Health
        │
        ├── Interaction recency
        ├── Interaction frequency
        ├── Interaction quality
        ├── Relationship history
        ├── User-defined importance
        └── Mutual interaction
```

---

## 🔔 Relationship Reminders

The application can notify users when a relationship may need attention.

Example:

> **Paul may be worth reconnecting with.**
>
> You normally interact every 30–45 days, but it has been 61 days since your last interaction.

The system should avoid treating every missed interaction as a problem.

Users should be able to define their own preferences.

---

## 💡 Reconnection Suggestions

Instead of simply saying:

> "Contact Paul."

the system can provide contextual suggestions.

For example:

> You last spoke with Paul about his startup.
>
> Consider asking how the project is progressing.

Possible suggestions:

* Send a message
* Make a phone call
* Invite them for coffee
* Share something relevant
* Congratulate them on an achievement
* Ask about something discussed previously
* Meet at an upcoming event

---

# Example User Journey

Consider John.

### Step 1 — John knows Jason

```text
John ───── Jason
```

Jason is John's college friend.

---

### Step 2 — John meets Paul

John attends a party at Jason's house and meets Jason's older brother Paul.

```text
John ───── Jason
            │
            └──── Paul
```

The system records:

```text
Paul
Met through: Jason
First met: Party at Jason's house
```

---

### Step 3 — John decides to maintain the relationship

John adds Paul as a relationship.

```text
John ───── Paul
```

The relationship is initially categorized as:

```text
Type: Acquaintance
Importance: Medium
```

---

### Step 4 — The relationship develops

John and Paul meet several times and exchange messages.

The relationship can evolve:

```text
Acquaintance
      ↓
Friend
      ↓
Professional Connection
```

The application maintains the history of this relationship.

---

### Step 5 — The relationship becomes inactive

After several months:

```text
Last interaction: 78 days ago
Typical interaction frequency: 30–45 days
```

The system identifies the relationship as potentially needing attention.

---

### Step 6 — Reconnection

The application recommends:

```text
Reconnect with Paul

Last conversation:
Paul was working on a startup.

Suggested action:
Ask how the startup is progressing.
```

John can then record the interaction.

The relationship health improves.

---

# Relationship Philosophy

Relationship Graph is **not intended to encourage transactional networking**.

A person should not be valuable simply because they have a particular job, social status, or professional position.

Relationships can have different forms of value:

```text
Friendship
Family
Emotional Support
Learning
Mentorship
Professional Growth
Community
Shared Interests
Recreation
Personal Development
```

The user decides why a relationship matters.

The goal is:

> **Maintain meaningful relationships, not simply collect more connections.**

---

# MVP

The initial version will focus on the fundamental relationship management features.

### People

* [ ] Create person
* [ ] Update person
* [ ] Delete person
* [ ] View person
* [ ] Search people

### Relationships

* [ ] Create relationship
* [ ] Define relationship type
* [ ] Set relationship importance
* [ ] Record how people met
* [ ] Record relationship start date
* [ ] Define preferred interaction frequency

### Interactions

* [ ] Record interaction
* [ ] View interaction history
* [ ] Add interaction notes
* [ ] Categorize interaction type

### Dashboard

* [ ] Relationships needing attention
* [ ] Recently interacted people
* [ ] Inactive relationships
* [ ] Upcoming important dates

### Notifications

* [ ] Relationship maintenance reminders
* [ ] Configurable reminder frequency

---

# Future Features

The project can gradually evolve into a more intelligent relationship management platform.

## Relationship Graph Visualization

Interactive visualization of the user's network.

```text
                 Paul
                  │
                  │
John ─────────── Jason ───────── Sarah
 │                │
 │                │
Mike             David
```

Users could explore how people are connected.

---

## Connection Discovery

Given a target person or topic, the system could identify possible connection paths.

Example:

```text
You want to meet someone
working in startup fundraising.

Possible connection:

You
 ↓
Jason
 ↓
Paul
 ↓
Michael
 ↓
Startup Investor
```

---

## Smart Relationship Scoring

A more advanced scoring system could estimate relationship health using historical interaction data.

```text
Relationship Health = f(
    Recency,
    Frequency,
    Interaction Quality,
    Importance,
    History,
    User Preferences
)
```

---

## AI-Assisted Suggestions

AI could eventually help with:

* Reconnection suggestions
* Conversation ideas
* Summarizing interaction history
* Identifying relationship patterns
* Generating personalized reminders
* Finding relevant connections
* Suggesting ways to maintain relationships

Example:

> "You haven't spoken with Sarah in two months. Last time you talked, she mentioned she was preparing for a design conference. Consider asking how it went."

---

## Important Events

Track events that matter to relationships.

Examples:

* Birthdays
* Anniversaries
* Graduation
* New job
* Promotion
* Important projects
* Personal milestones

The system can provide timely reminders.

---

# Domain Model

A possible initial domain model:

```text
┌──────────────┐
│    Person    │
└──────┬───────┘
       │
       │ has
       ▼
┌──────────────┐
│ Relationship │
└──────┬───────┘
       │
       ├───────────────┐
       │               │
       ▼               ▼
┌──────────────┐ ┌──────────────┐
│ Interaction  │ │ Relationship │
│              │ │    Type      │
└──────────────┘ └──────────────┘
```

Potential domain entities:

```text
Person
Relationship
Interaction
RelationshipType
RelationshipGoal
Reminder
ImportantEvent
```

Potential value objects:

```text
RelationshipStrength
RelationshipHealth
InteractionFrequency
ContactMethod
```

---

# Architecture

The architecture will evolve as the domain becomes better understood.

A possible backend structure:

```text
API
 │
 ▼
Controller
 │
 ▼
Application Service
 │
 ▼
Domain
 │
 ├── Entities
 ├── Value Objects
 ├── Domain Services
 └── Domain Events
 │
 ▼
Infrastructure
 │
 ├── Database
 ├── Notifications
 └── External Services
```

The project is intended to be a practical environment for exploring:

* Domain-Driven Design
* Clean Architecture
* SOLID principles
* Design Patterns
* REST APIs
* Background Jobs
* Event-Driven Architecture
* Graph Data Modeling
* Recommendation Systems
* AI-assisted applications

---

# Design Principles

### 1. Relationships are more than contacts

A contact contains information about a person.

A relationship contains **context**.

---

### 2. Time alone should not define relationship strength

Not speaking to someone for 60 days doesn't necessarily mean the relationship is weak.

The system should consider context and history.

---

### 3. The user owns the relationship data

Relationship information should be private by default.

The application should never assume that another person knows how the user has categorized or evaluated their relationship.

---

### 4. Meaningful relationships over network size

The goal isn't:

```text
500 connections
```

The goal is:

```text
Relationships worth maintaining
```

---

# Technology Stack

> This section can be updated as implementation decisions are made.

Possible stack:

### Backend

* Go / Python
* REST API
* PostgreSQL

### Frontend

* React / Next.js
* TypeScript

### Infrastructure

* Docker
* PostgreSQL
* Redis
* Background workers

### Future

* Graph database or graph-oriented queries
* AI/LLM integration
* Push notifications
* Email notifications

---

# Development Roadmap

## Phase 1 — Foundation

```text
[ ] Project setup
[ ] Database design
[ ] Person management
[ ] Relationship management
[ ] Interaction management
```

## Phase 2 — Relationship Management

```text
[ ] Relationship health
[ ] Maintenance rules
[ ] Reminder system
[ ] Dashboard
[ ] Important events
```

## Phase 3 — Relationship Graph

```text
[ ] Connection graph
[ ] Mutual connections
[ ] Connection paths
[ ] Graph visualization
```

## Phase 4 — Intelligence

```text
[ ] Smart relationship scoring
[ ] Reconnection recommendations
[ ] Personalized suggestions
[ ] AI-assisted interaction summaries
```

## Phase 5 — Automation

```text
[ ] Scheduled notifications
[ ] Calendar integration
[ ] Contact synchronization
[ ] Communication integrations
```

---

# Example

A user's dashboard might eventually look like:

```text
┌─────────────────────────────────────────────────┐
│                 Good morning!                   │
│                                                 │
│  Relationships needing attention                │
│                                                 │
│  🔴 Paul                                        │
│     Last interaction: 61 days ago               │
│     Typical frequency: 30–45 days               │
│                                                 │
│     "You discussed his startup last time."      │
│                                                 │
│     [ Reconnect ]                               │
│                                                 │
│  🟡 Sarah                                       │
│     Last interaction: 38 days ago               │
│                                                 │
│     [ View relationship ]                       │
│                                                 │
│  🟢 Jason                                       │
│     Relationship healthy                        │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

# Project Goal

Relationship Graph aims to answer three fundamental questions:

### 1. Who are the people in my network?

```text
People + Connections
```

### 2. Which relationships need attention?

```text
History + Interaction + Preferences
```

### 3. What can I do to maintain them meaningfully?

```text
Context + Recommendations + Reminders
```

Ultimately:

> **Relationship Graph is a personal system for understanding and intentionally maintaining the relationships that matter.**

---

# Status

🚧 **Early Development / Concept**

The domain model, architecture, technology choices, and feature set are expected to evolve as the project is developed and tested with real use cases.

---

# License

License to be determined.

