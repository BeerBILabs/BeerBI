import UsersPage from '../components/UsersPage'
import type { ReactElement } from 'react'

export default function Home(): ReactElement {
  return (
    <main className="flex-1 flex flex-col items-center justify-center px-2">
      <div className="w-full max-w-3xl card rounded-2xl shadow-lg p-8 mt-10 mb-10 border bordered">
        <UsersPage />
      </div>
    </main>
  )
}
