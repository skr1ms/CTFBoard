/**
 * Central registry of data-testid values.
 * Usage: page.getByTestId(tid.login.submit)
 */
export const tid = {
  // Auth
  login: {
    email: 'login-email',
    password: 'login-password',
    submit: 'login-submit',
    error: 'login-error',
  },
  register: {
    username: 'register-username',
    email: 'register-email',
    password: 'register-password',
    confirm: 'register-confirm',
    submit: 'register-submit',
    error: 'register-error',
  },
  resetPassword: {
    email: 'reset-email',
    submit: 'reset-submit',
    password: 'reset-password',
    confirm: 'reset-confirm',
    submitNew: 'reset-submit-new',
  },

  // Challenges
  challengeCard: 'challenge-card',
  challengeModal: {
    root: 'challenge-modal',
    close: 'challenge-modal-close',
    tab: (name: string) => `challenge-modal-tab-${name}`,
  },
  flagSubmit: {
    input: 'flag-submit-input',
    button: 'flag-submit-button',
    status: 'flag-submit-status',
  },
  hint: {
    unlock: (id: string | number) => `hint-unlock-${id}`,
    confirmYes: 'hint-confirm-yes',
    content: (id: string | number) => `hint-content-${id}`,
  },

  // Team
  teamEnroll: {
    createName: 'team-create-name',
    createSubmit: 'team-create-submit',
    joinToken: 'team-join-token',
    joinSubmit: 'team-join-submit',
    soloSubmit: 'team-solo-submit',
  },
  myTeam: {
    nameEdit: 'team-name-edit',
    inviteCopy: 'team-invite-copy',
    inviteRegen: 'team-invite-regen',
    memberRow: (userId: string) => `team-member-row-${userId}`,
    kick: (userId: string) => `team-kick-${userId}`,
    leave: 'team-leave',
    disband: 'team-disband',
  },

  // Settings
  settings: {
    username: 'settings-username',
    email: 'settings-email',
    profileSave: 'settings-profile-save',
    passwordCurrent: 'settings-password-current',
    passwordNew: 'settings-password-new',
    passwordSave: 'settings-password-save',
    avatarUpload: 'settings-avatar-upload',
    avatarDelete: 'settings-avatar-delete',
    tokenCreate: 'settings-token-create',
    tokenRow: (id: string) => `settings-token-row-${id}`,
  },

  // Notifications
  notifications: {
    tabPersonal: 'notifications-tab-personal',
    tabGlobal: 'notifications-tab-global',
    item: (id: string) => `notification-item-${id}`,
  },

  // Scoreboard
  scoreboard: {
    row: (teamId: string) => `scoreboard-row-${teamId}`,
    graph: 'scoreboard-graph',
  },

  // Navbar
  navbar: {
    logout: 'navbar-logout',
    userMenuTrigger: 'navbar-user-menu-trigger',
    notificationsBell: 'navbar-notifications-bell',
    unreadBadge: 'navbar-unread-badge',
  },

  // Admin
  admin: {
    row: (id: string) => `admin-row-${id}`,
    delete: (id: string) => `admin-delete-${id}`,
    edit: (id: string) => `admin-edit-${id}`,
    bulkDelete: 'admin-bulk-delete',
  },

  // Pagination
  pagination: {
    prev: 'pagination-prev',
    next: 'pagination-next',
    page: (n: number) => `pagination-page-${n}`,
  },
} as const
