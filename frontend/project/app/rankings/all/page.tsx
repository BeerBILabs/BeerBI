import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'All Time Rankings | BeerBot',
};

export default function AllTimeRankingsPage() {
  return (
    <div>
      <h1
        className="text-2xl font-bold mb-4"
        style={{ color: 'hsl(var(--primary))' }}
      >
        All Time Rankings
      </h1>
      <p className="text-muted-foreground">
        All-time leaderboard coming soon...
      </p>
    </div>
  );
}
