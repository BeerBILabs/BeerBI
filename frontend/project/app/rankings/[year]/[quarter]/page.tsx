import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import { isValidQuarter, formatQuarterLabel } from '../../../../lib/quarters';
import { QuarterlyLeaderboard } from '@/components/QuarterlyLeaderboard';

interface PageProps {
  params: Promise<{ year: string; quarter: string }>;
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { year, quarter } = await params;

  // Extract quarter number from q1, q2, etc.
  const quarterMatch = quarter.match(/^q([1-4])$/i);
  if (!quarterMatch) {
    return { title: 'Not Found | BeerBot' };
  }

  const quarterNum = parseInt(quarterMatch[1], 10);
  return {
    title: `${formatQuarterLabel(quarterNum)} ${year} Rankings | BeerBot`,
  };
}

export default async function QuarterlyRankingsPage({ params }: PageProps) {
  const { year, quarter } = await params;

  // Validate year format: 4-digit number in range 2020-2099
  const yearNum = parseInt(year, 10);
  if (!/^\d{4}$/.test(year) || yearNum < 2020 || yearNum > 2099) {
    notFound();
  }

  // Validate quarter format: q1, q2, q3, or q4 (case insensitive)
  const quarterMatch = quarter.match(/^q([1-4])$/i);
  if (!quarterMatch) {
    notFound();
  }

  const quarterNum = parseInt(quarterMatch[1], 10);

  // Check if quarter is valid (not in the future)
  if (!isValidQuarter(yearNum, quarterNum)) {
    notFound();
  }

  return <QuarterlyLeaderboard year={yearNum} quarter={quarterNum} showRankChange={true} />;
}
