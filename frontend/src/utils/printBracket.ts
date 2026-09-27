import type { Match, Athlete } from "../types";

export const COURT_LABELS = [
  "A",
  "B",
  "C",
  "D",
  "E",
  "F",
  "G",
  "H",
];

export function courtLabel(n: number): string {
  return COURT_LABELS[(n - 1) % COURT_LABELS.length] ?? String(n);
}

interface RoundColumn {
  round: number;
  matches: Match[];
}

export function buildBracketColumns(matches: Match[]): {
  left: RoundColumn[];
  final: Match | null;
  right: RoundColumn[];
} {
  if (matches.length === 0) {
    return {
      left: [],
      final: null,
      right: [],
    };
  }

  const maxRound = Math.max(...matches.map((match) => match.round));

  const left = Array.from(
      new Set(
          matches
              .filter((match) => match.side === "left")
              .map((match) => match.round),
      ),
  )
      .sort((a, b) => a - b)
      .map((round) => ({
        round,
        matches: matches.filter(
            (match) => match.side === "left" && match.round === round,
        ),
      }));

  const right = Array.from(
      new Set(
          matches
              .filter((match) => match.side === "right")
              .map((match) => match.round),
      ),
  )
      .sort((a, b) => b - a)
      .map((round) => ({
        round,
        matches: matches.filter(
            (match) => match.side === "right" && match.round === round,
        ),
      }));

  const final =
      matches.find(
          (match) => match.side === "final" && match.round === maxRound,
      ) ?? null;

  return {
    left,
    final,
    right,
  };
}

export function athleteName(
    athletes: Athlete[],
    id: string | null | undefined,
): string {
  if (!id) {
    return "-";
  }

  return athletes.find((athlete) => athlete.id === id)?.name ?? "-";
}
