export type CompetitionMode = 'teams_only' | 'solo_only'

export function normalizeCompetitionMode(mode?: string): CompetitionMode {
  if (mode === 'solo_only' || mode === 'teams_only') {
    return mode
  }

  return 'teams_only'
}

export function participantCopy(mode?: string) {
  switch (normalizeCompetitionMode(mode)) {
    case 'solo_only':
      return {
        singular: 'Player',
        plural: 'Players',
        singularLower: 'player',
        pluralLower: 'players',
        unknown: 'Unknown player',
      }
    case 'teams_only':
    default:
      return {
        singular: 'Team',
        plural: 'Teams',
        singularLower: 'team',
        pluralLower: 'teams',
        unknown: 'Unknown team',
      }
  }
}
