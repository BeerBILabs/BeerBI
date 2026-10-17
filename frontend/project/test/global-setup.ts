// Runs once in the main Vitest process before workers start; workers inherit the env.
// Pinning the timezone makes timezone bugs (e.g. toISOString() on local midnight)
// show up on UTC CI runners as well as on developer machines.
export default function setup() {
  process.env.TZ = 'Europe/Berlin';
}
