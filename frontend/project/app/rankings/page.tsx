import { redirect } from 'next/navigation';
import { getCurrentQuarter } from '../../lib/quarters';

export default function RankingsPage() {
  const { year, quarter } = getCurrentQuarter();
  redirect(`/rankings/${year}/q${quarter}`);
}
