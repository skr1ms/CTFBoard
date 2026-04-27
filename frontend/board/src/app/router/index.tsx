import { AdminGuard } from '@/app/guards/AdminGuard'
import { AuthGuard } from '@/app/guards/AuthGuard'
import { GuestGuard } from '@/app/guards/GuestGuard'
import { SetupCompleteGuard, SetupGuard } from '@/app/guards/SetupGuard'
import { TeamGuard } from '@/app/guards/TeamGuard'
import { AdminLayout } from '@/app/layouts/AdminLayout'
import { AuthLayout } from '@/app/layouts/AuthLayout'
import { ErrorLayout } from '@/app/layouts/ErrorLayout'
import { PublicLayout } from '@/app/layouts/PublicLayout'
import { PublicSetupLayout } from '@/app/layouts/PublicSetupLayout'
import { useAuthStore } from '@/shared/stores/authStore'
import { ErrorBoundary } from '@/shared/ui/error-boundary'
import { Spinner } from '@/shared/ui/spinner'
import { lazy, Suspense } from 'react'
import { BrowserRouter, Navigate, Outlet, Route, Routes } from 'react-router'

const OAuthCallbackPage = lazy(() =>
  import('@/pages/auth/oauth-callback').then((m) => ({ default: m.OAuthCallbackPage })),
)
const LoginPage = lazy(() => import('@/pages/login').then((m) => ({ default: m.LoginPage })))
const RegisterPage = lazy(() =>
  import('@/pages/register').then((m) => ({ default: m.RegisterPage })),
)
const ResetPasswordPage = lazy(() =>
  import('@/pages/reset-password').then((m) => ({ default: m.ResetPasswordPage })),
)
const VerifyEmailPage = lazy(() =>
  import('@/pages/verify-email').then((m) => ({ default: m.VerifyEmailPage })),
)

const ChallengesPage = lazy(() =>
  import('@/pages/challenges').then((m) => ({ default: m.ChallengesPage })),
)
const ScoreboardPage = lazy(() =>
  import('@/pages/scoreboard').then((m) => ({ default: m.ScoreboardPage })),
)
const SettingsPage = lazy(() =>
  import('@/pages/settings').then((m) => ({ default: m.SettingsPage })),
)
const ActivityPage = lazy(() =>
  import('@/pages/activity').then((m) => ({ default: m.ActivityPage })),
)
const NotificationsPage = lazy(() =>
  import('@/pages/notifications').then((m) => ({ default: m.NotificationsPage })),
)
const StaticPage = lazy(() =>
  import('@/pages/static-page').then((m) => ({ default: m.StaticPage })),
)
const TeamPage = lazy(() => import('@/pages/team').then((m) => ({ default: m.TeamPage })))
const TeamEnrollPage = lazy(() =>
  import('@/pages/team/enroll').then((m) => ({ default: m.TeamEnrollPage })),
)
const UsersPage = lazy(() => import('@/pages/users').then((m) => ({ default: m.UsersPage })))
const UserProfilePage = lazy(() =>
  import('@/pages/users').then((m) => ({ default: m.UserProfilePage })),
)
const TeamsPage = lazy(() => import('@/pages/teams').then((m) => ({ default: m.TeamsPage })))
const TeamPublicProfilePage = lazy(() =>
  import('@/pages/teams').then((m) => ({ default: m.TeamPublicProfilePage })),
)

const AdminStatisticsPage = lazy(() =>
  import('@/pages/admin/statistics').then((m) => ({ default: m.AdminStatisticsPage })),
)
const AdminChallengesPage = lazy(() =>
  import('@/pages/admin/challenges').then((m) => ({ default: m.AdminChallengesPage })),
)
const AdminChallengeDetailPage = lazy(() =>
  import('@/pages/admin/challenges/detail').then((m) => ({ default: m.AdminChallengeDetailPage })),
)
const AdminUsersPage = lazy(() =>
  import('@/pages/admin/users').then((m) => ({ default: m.AdminUsersPage })),
)
const AdminUserDetailPage = lazy(() =>
  import('@/pages/admin/users/detail').then((m) => ({ default: m.AdminUserDetailPage })),
)
const AdminTeamsPage = lazy(() =>
  import('@/pages/admin/teams').then((m) => ({ default: m.AdminTeamsPage })),
)
const AdminTeamDetailPage = lazy(() =>
  import('@/pages/admin/teams/detail').then((m) => ({ default: m.AdminTeamDetailPage })),
)
const AdminSubmissionsPage = lazy(() =>
  import('@/pages/admin/submissions').then((m) => ({ default: m.AdminSubmissionsPage })),
)
const AdminAwardsPage = lazy(() =>
  import('@/pages/admin/awards').then((m) => ({ default: m.AdminAwardsPage })),
)
const AdminCompetitionPage = lazy(() =>
  import('@/pages/admin/competition').then((m) => ({ default: m.AdminCompetitionPage })),
)
const AdminSettingsPage = lazy(() =>
  import('@/pages/admin/settings').then((m) => ({ default: m.AdminSettingsPage })),
)
const AdminConfigsPage = lazy(() =>
  import('@/pages/admin/configs').then((m) => ({ default: m.AdminConfigsPage })),
)
const AdminScoreboardPage = lazy(() =>
  import('@/pages/admin/scoreboard').then((m) => ({ default: m.AdminScoreboardPage })),
)
const AdminNotificationsPage = lazy(() =>
  import('@/pages/admin/notifications').then((m) => ({ default: m.AdminNotificationsPage })),
)
const AdminPagesPage = lazy(() =>
  import('@/pages/admin/pages').then((m) => ({ default: m.AdminPagesPage })),
)
const AdminTagsPage = lazy(() =>
  import('@/pages/admin/tags').then((m) => ({ default: m.AdminTagsPage })),
)
const AdminBracketsPage = lazy(() =>
  import('@/pages/admin/brackets').then((m) => ({ default: m.AdminBracketsPage })),
)
const AdminBackupPage = lazy(() =>
  import('@/pages/admin/backup').then((m) => ({ default: m.AdminBackupPage })),
)
const AdminFieldsPage = lazy(() =>
  import('@/pages/admin/fields').then((m) => ({ default: m.AdminFieldsPage })),
)
const AdminUnlocksPage = lazy(() =>
  import('@/pages/admin/unlocks').then((m) => ({ default: m.AdminUnlocksPage })),
)
const AdminAppealsPage = lazy(() =>
  import('@/pages/admin/appeals').then((m) => ({ default: m.AdminAppealsPage })),
)
const AdminStoragePage = lazy(() =>
  import('@/pages/admin/storage').then((m) => ({ default: m.AdminStoragePage })),
)

const SetupPage = lazy(() => import('@/pages/setup').then((m) => ({ default: m.SetupPage })))

const TosPage = lazy(() => import('@/pages/tos').then((m) => ({ default: m.TosPage })))
const PrivacyPage = lazy(() => import('@/pages/privacy').then((m) => ({ default: m.PrivacyPage })))
const BannedPage = lazy(() => import('@/pages/banned').then((m) => ({ default: m.BannedPage })))

const Page403 = lazy(() => import('@/pages/errors/403').then((m) => ({ default: m.Page403 })))
const Page404 = lazy(() => import('@/pages/errors/404').then((m) => ({ default: m.Page404 })))
const Page429 = lazy(() => import('@/pages/errors/429').then((m) => ({ default: m.Page429 })))
const Page500 = lazy(() => import('@/pages/errors/500').then((m) => ({ default: m.Page500 })))

function PageFallback() {
  return (
    <div className="flex items-center justify-center min-h-[40vh]">
      <Spinner size="lg" />
    </div>
  )
}

function RootRedirect() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return <Navigate to={isAuthenticated ? '/challenges' : '/login'} replace />
}

export function AppRouter() {
  return (
    <BrowserRouter>
      <ErrorBoundary>
        <Suspense fallback={<PageFallback />}>
          <Routes>
            {/* OAuth callback - outside all guards (must be reachable before setup) */}
            <Route path="/auth/callback" element={<OAuthCallbackPage />} />
            {/* Setup wizard - only accessible when setup is NOT complete */}
            <Route
              path="/setup"
              element={
                <SetupCompleteGuard>
                  <PublicSetupLayout>
                    <SetupPage />
                  </PublicSetupLayout>
                </SetupCompleteGuard>
              }
            />
            {/* All other routes are gated: redirect to /setup until setup_complete=true */}
            <Route element={<SetupGuard />}>
              {/* Root redirect */}
              <Route path="/" element={<RootRedirect />} />

              {/* Guest-only routes */}
              <Route
                element={
                  <PublicLayout>
                    <GuestGuard>
                      <Outlet />
                    </GuestGuard>
                  </PublicLayout>
                }
              >
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
                <Route path="/reset-password" element={<ResetPasswordPage />} />
              </Route>

              {/* Public routes (accessible without auth) */}
              <Route
                element={
                  <PublicLayout>
                    <Outlet />
                  </PublicLayout>
                }
              >
                <Route path="/verify-email" element={<VerifyEmailPage />} />
                <Route path="/scoreboard" element={<ScoreboardPage />} />
                <Route path="/teams" element={<TeamsPage />} />
                <Route path="/teams/:id" element={<TeamPublicProfilePage />} />
                <Route path="/users" element={<UsersPage />} />
                <Route path="/users/:id" element={<UserProfilePage />} />
                <Route path="/pages/:slug" element={<StaticPage />} />
                <Route path="/tos" element={<TosPage />} />
                <Route path="/privacy" element={<PrivacyPage />} />
              </Route>

              {/* Banned landing - requires auth to prevent guest->401->logout loop */}
              <Route
                path="/banned"
                element={
                  <AuthGuard>
                    <BannedPage />
                  </AuthGuard>
                }
              />

              {/* Authenticated routes */}
              <Route
                element={
                  <AuthLayout>
                    <AuthGuard>
                      <Outlet />
                    </AuthGuard>
                  </AuthLayout>
                }
              >
                <Route path="/challenges" element={<ChallengesPage />} />
                <Route path="/challenges/:id" element={<ChallengesPage />} />
                <Route path="/notifications" element={<NotificationsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/activity" element={<ActivityPage />} />
                <Route path="/team/enroll" element={<TeamEnrollPage />} />
                <Route
                  path="/team"
                  element={
                    <TeamGuard>
                      <TeamPage />
                    </TeamGuard>
                  }
                />
              </Route>

              {/* Admin routes */}
              <Route
                element={
                  <AuthLayout>
                    <AuthGuard>
                      <AdminGuard>
                        <AdminLayout>
                          <Outlet />
                        </AdminLayout>
                      </AdminGuard>
                    </AuthGuard>
                  </AuthLayout>
                }
              >
                <Route path="/admin" element={<Navigate to="/admin/statistics" replace />} />
                <Route path="/admin/statistics" element={<AdminStatisticsPage />} />
                <Route path="/admin/challenges" element={<AdminChallengesPage />} />
                <Route path="/admin/challenges/:id" element={<AdminChallengeDetailPage />} />
                <Route path="/admin/users" element={<AdminUsersPage />} />
                <Route path="/admin/users/:id" element={<AdminUserDetailPage />} />
                <Route path="/admin/teams" element={<AdminTeamsPage />} />
                <Route path="/admin/teams/:id" element={<AdminTeamDetailPage />} />
                <Route path="/admin/submissions" element={<AdminSubmissionsPage />} />
                <Route path="/admin/awards" element={<AdminAwardsPage />} />
                <Route path="/admin/competition" element={<AdminCompetitionPage />} />
                <Route path="/admin/settings" element={<AdminSettingsPage />} />
                <Route path="/admin/configs" element={<AdminConfigsPage />} />
                <Route path="/admin/scoreboard" element={<AdminScoreboardPage />} />
                <Route path="/admin/notifications" element={<AdminNotificationsPage />} />
                <Route path="/admin/pages" element={<AdminPagesPage />} />
                <Route path="/admin/tags" element={<AdminTagsPage />} />
                <Route path="/admin/brackets" element={<AdminBracketsPage />} />
                <Route path="/admin/backup" element={<AdminBackupPage />} />
                <Route path="/admin/fields" element={<AdminFieldsPage />} />
                <Route path="/admin/unlocks" element={<AdminUnlocksPage />} />
                <Route path="/admin/appeals" element={<AdminAppealsPage />} />
                <Route path="/admin/storage" element={<AdminStoragePage />} />
              </Route>

              {/* Error pages */}
              <Route
                path="/403"
                element={
                  <ErrorLayout>
                    <Page403 />
                  </ErrorLayout>
                }
              />
              <Route
                path="/429"
                element={
                  <ErrorLayout>
                    <Page429 />
                  </ErrorLayout>
                }
              />
              <Route
                path="/500"
                element={
                  <ErrorLayout>
                    <Page500 />
                  </ErrorLayout>
                }
              />
              <Route
                path="*"
                element={
                  <ErrorLayout>
                    <Page404 />
                  </ErrorLayout>
                }
              />
            </Route>{' '}
            {/* end SetupGuard */}
          </Routes>
        </Suspense>
      </ErrorBoundary>
    </BrowserRouter>
  )
}
