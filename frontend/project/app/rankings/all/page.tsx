import type { Metadata } from 'next';
import { QuarterlyLeaderboard } from '@/components/QuarterlyLeaderboard';

export const metadata: Metadata = {
  title: 'All Time Rankings | BeerBot',
};

export default function AllTimeRankingsPage() {
  return <QuarterlyLeaderboard key="all-time" year={null} quarter={null} showRankChange={false} />;
}
