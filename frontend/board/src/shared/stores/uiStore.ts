import { create } from 'zustand'

export interface UiState {
  sidebarCollapsed: boolean
  mobileMenuOpen: boolean

  toggleSidebar(): void
  setSidebarCollapsed(collapsed: boolean): void
  toggleMobileMenu(): void
  setMobileMenuOpen(open: boolean): void
}

export const useUiStore = create<UiState>((set) => ({
  sidebarCollapsed: false,
  mobileMenuOpen: false,

  toggleSidebar() {
    set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed }))
  },

  setSidebarCollapsed(collapsed) {
    set({ sidebarCollapsed: collapsed })
  },

  toggleMobileMenu() {
    set((s) => ({ mobileMenuOpen: !s.mobileMenuOpen }))
  },

  setMobileMenuOpen(open) {
    set({ mobileMenuOpen: open })
  },
}))
