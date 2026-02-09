import { useState, useEffect, useCallback } from 'react'
import type { User } from '../types'
import { listUsers, approveUser, deleteUser } from '../lib/api'
import { Layout } from '../components/Layout'

export function AdminUsers() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      const data = await listUsers()
      setUsers(data)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const [busyUsers, setBusyUsers] = useState<Record<number, 'approving' | 'deleting'>>({})

  const handleApprove = async (id: number) => {
    if (busyUsers[id]) return
    setBusyUsers((prev) => ({ ...prev, [id]: 'approving' }))
    try {
      await approveUser(id)
      await refresh()
    } catch (err) {
      alert('Approve failed: ' + (err as Error).message)
    } finally {
      setBusyUsers((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
    }
  }

  const handleDelete = async (id: number) => {
    if (busyUsers[id]) return
    if (!confirm('Delete this user? This cannot be undone.')) return
    setBusyUsers((prev) => ({ ...prev, [id]: 'deleting' }))
    try {
      await deleteUser(id)
      await refresh()
    } catch (err) {
      alert('Delete failed: ' + (err as Error).message)
    } finally {
      setBusyUsers((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
    }
  }

  return (
    <Layout>
      <div className="flex items-center gap-3 mb-4">
        <span className="font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-widest">
          // user management
        </span>
        {users.length > 0 && (
          <span className="font-mono text-[0.7rem] text-text-dim">
            [{users.length}]
          </span>
        )}
      </div>
      <div className="tech-panel p-5">
        {loading ? (
          <div className="text-text-tertiary text-sm py-10 text-center font-mono">loading...</div>
        ) : users.length === 0 ? (
          <div className="text-text-tertiary text-sm py-10 text-center font-mono">no users</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr>
                  {['address', 'role', 'status', 'created', ''].map((h) => (
                    <th
                      key={h}
                      className="text-left font-mono text-[0.7rem] font-medium text-text-tertiary uppercase tracking-wider px-3 pb-2 border-b border-border whitespace-nowrap"
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {users.map((u) => (
                  <tr key={u.id} className="group">
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono font-medium text-text whitespace-nowrap text-xs">
                      {u.address.slice(0, 6)}...{u.address.slice(-4)}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono text-[0.75rem] text-text-secondary whitespace-nowrap">
                      {u.role}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 whitespace-nowrap">
                      {u.approved ? (
                        <span className="inline-flex items-center gap-1.5 font-mono text-[0.7rem] text-accent-text">
                          <span className="w-1.5 h-1.5 rounded-full bg-accent" />
                          approved
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5 font-mono text-[0.7rem] text-warning">
                          <span className="w-1.5 h-1.5 rounded-full bg-warning animate-[pulse-dot_2s_ease-in-out_infinite]" />
                          pending
                        </span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 font-mono text-[0.75rem] text-text-tertiary whitespace-nowrap">
                      {new Date(u.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-3 py-2.5 border-b border-border/30 text-right whitespace-nowrap">
                      {busyUsers[u.id] ? (
                        <span className="font-mono text-[0.7rem] text-text-tertiary">
                          {busyUsers[u.id]}<span className="animate-[blink_1s_step-end_infinite]">_</span>
                        </span>
                      ) : (
                        <div className="flex gap-1.5 justify-end">
                          {!u.approved && (
                            <button
                              onClick={() => handleApprove(u.id)}
                              className="bg-accent/10 text-accent-text border border-accent/30 font-mono text-[0.7rem] font-medium rounded px-2 py-0.5 cursor-pointer hover:bg-accent/20 hover:border-accent/50 transition-colors"
                            >
                              approve
                            </button>
                          )}
                          <button
                            onClick={() => handleDelete(u.id)}
                            className="bg-transparent text-text-tertiary border border-border font-mono text-[0.7rem] px-2 py-0.5 rounded hover:text-danger hover:border-danger/40 hover:bg-danger-muted transition-colors cursor-pointer"
                          >
                            delete
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </Layout>
  )
}
