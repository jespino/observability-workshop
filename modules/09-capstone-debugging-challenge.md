# Module 9: Capstone — The Debugging Challenge

**Duration:** ~45 min (30 min investigating, 15 min debrief) · **Mixed teams across both tracks**

Everything you built is now a tool. Here's the test. 🕵️

## Learning Objectives

By the end of this module, you will:

- Have diagnosed an unfamiliar failure using only telemetry
- Know from experience which signal breaks a given class of problem open
- Have a concrete list of what you wish you'd instrumented

## Prerequisites

- Modules 0–8 completed
- Your dashboard, your logs panel and your four Grafana jumps working

## The setup

Your facilitator is about to restart `todo-api` as a "new release" and turn up the load. It takes about thirty seconds — one environment variable and a heavier load profile.

The only briefing you get is:

> **"Users are complaining the app is slow. Go."**

## The rules

1. **No reading application source until you can point at the problem in telemetry.** This is the whole exercise. Breaking this rule doesn't cheat us, it cheats you.
2. **Teams of 2–3, mixed across tracks.** Someone on JS and someone on Python looking at the same dashboard is the ideal pairing.
3. **Write as you go.** Keep a shared doc with four columns, and fill it in as you work:

   | Symptom | Signal that showed it | Hypothesis | Confirmation |
   |---|---|---|---|
   | | | | |

4. **There is more than one problem.** Don't stop at the first thing you find.
5. Ten minutes in, if you have nothing, your facilitator will give a nudge. Take it — there's no prize for suffering.

## Where to start

If you're staring at a blank screen, here's a sequence that works. It's the same sequence you'd use on a real system.

1. **Confirm the symptom is real, in metrics.** Is latency actually up? For all endpoints or some? Since when?

   ```
   histogram_quantile(0.95, sum(rate(http_server_duration_milliseconds_bucket[5m])) by (le, http_route))
   ```

2. **Narrow to a route, then to an attribute.** Which route degraded? Does it correlate with a `list`, a `priority`, a client?

3. **Look for errors, separately from latency.** They're often two different problems and the loud one hides the quiet one.

   ```
   service.name:todo-api AND severity_text:ERROR AND _time:30m | stats by (body) count()
   ```

4. **Go to traces for the slow requests.** Not the average one — a slow one.

   ```
   _time:30m "resource_attr:service.name":=todo-api duration:>1s
   ```

   Then ask of the waterfall: is one span slow, or are there *many* spans? Those are completely different diseases.

5. **Check your business metrics.** The ones you built in Module 5 that nobody else would have. Are any of them behaving strangely in a way the HTTP metrics don't show?

6. **Check the pipeline itself.** If something looks impossible, make sure you're not looking at a Collector problem (Module 4.2).

## What you're looking for

Two independent faults are live. They're deliberately chosen to be found by different signals, so a team that only looks at one dashboard will find one and miss the other.

Without giving anything away:

- One of them is **invisible in aggregate metrics** because it affects a minority of requests. Averages hide it. You'll only see it if you look at an individual trace, or if you slice by the right attribute.
- The other is **completely invisible in traces**, and not because it is fast — because it never touches the network at all. No library was involved, so no library instrumented it. Only a metric you built yourself in Module 5 shows it exists.

That second one is the sharper lesson. A trace shows you what crossed a boundary. Work
that stays inside your process is invisible unless *you* made it visible. **Every signal
has a blind spot, and they're not the same blind spot.**

## Debrief [~15 min]

This is the most important conversation of the workshop. Don't let it get cut for time.

Go around the room:

1. **Which signal broke it open for you?** Different teams will answer differently for the same fault, and that's interesting in itself.
2. **What did you wish you'd instrumented in Module 5 that you didn't?** Be specific. This is the single most valuable question of the day.
3. **What did you build that turned out to be useless?** Equally valuable, and much less often asked.
4. **What would you add to your own service on Monday?** One thing. Concrete. Write it down.

Write the collective answer on the wall and leave it there for the rest of the session.

## Think about

You found the problems using a system you built in one day, on an application you'd never seen before this morning. The system you actually work on is bigger and you know it better.

So: what's the actual blocker to doing this at work? Is it tooling, or is it something else?

## If you're stuck

- **Everything looks normal.** Widen your time range to before the deploy and compare. A metric that looks fine in isolation often looks obviously wrong next to yesterday.
- **You found one fault and the room is still going.** Go back to step 5. The second fault is not in the same signal as the first.
- **You want to read the source.** Not yet. Ten more minutes. 😉

## What We Accomplished

- A real diagnosis, from symptom to root cause, using only telemetry
- Direct experience of each signal's blind spot
- A concrete, personal list of what to instrument next

## Next Steps

One short conversation left, and it's the one about what happens when this leaves your laptop. Continue with [Module 10: Production Realities and Wrap-Up](./10-production-realities.md).
