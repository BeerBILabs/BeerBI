import type { Metadata } from 'next';
import { Leaderboard } from '@/components/Leaderboard';

export const metadata: Metadata = {
  title: 'All Time Rankings | BeerBot',
};

export default function AllTimeRankingsPage() {
  return <Leaderboard key="all-time" year={null} quarter={null} showRankChange={false} />;
}
