import type { ReactNode } from 'react';

interface RankingsLayoutProps {
  children: ReactNode;
}

export default function RankingsLayout({ children }: RankingsLayoutProps) {
  return (
    <div className="rankings-container">
      {/* Tab navigation will be added in Plan 02 */}
      {children}
    </div>
  );
}
