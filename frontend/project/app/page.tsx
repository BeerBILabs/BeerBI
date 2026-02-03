import UsersPage from '../components/UsersPage'
import { QuarterlyRankingsPanel } from '../components/QuarterlyRankingsPanel'
import type { ReactElement } from 'react'

export default function Home(): ReactElement {
  return (
    <main className="flex-1 flex flex-col items-center px-8">
      <div className="w-full max-w-full grid grid-cols-1 lg:grid-cols-2 gap-6 mt-10 mb-10">
        <div className="card rounded-2xl shadow-lg p-8 border bordered">
          <UsersPage />
        </div>
        <div className="card rounded-2xl shadow-lg p-8 border bordered">
          <QuarterlyRankingsPanel />
        </div>
      </div>
    </main>
  )
}
