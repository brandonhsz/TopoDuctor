import React from "react";
import { Box, Text } from "ink";
import type { Worktree } from "../git/porcelain.js";
import { gridCols } from "./gridNav.js";
import {
  cardBranchText,
  cardNameText,
  cardOuterW,
  gridColGap,
  gridTotalWidth,
} from "./cardText.js";

type Props = {
  worktrees: Worktree[];
  cursor: number;
  termW: number;
  showSelection: boolean;
  marqueePhase: number;
  activeProject: string;
  /** Faint backdrop when a modal is open (Bubble Tea lipgloss.Faint parity). */
  dimmed?: boolean;
};

function WorktreeCard({
  wt,
  selected,
  marqueePhase,
  dimmed,
}: {
  wt: Worktree;
  selected: boolean;
  marqueePhase: number;
  dimmed?: boolean;
}) {
  const name = cardNameText(wt, selected, marqueePhase);
  const branch = cardBranchText(wt, selected, marqueePhase);
  const border =
    dimmed ? "gray" : selected ? "magenta" : "gray";
  return (
    <Box
      flexDirection="column"
      borderStyle="round"
      borderColor={border}
      width={cardOuterW}
      paddingX={1}
    >
      <Text bold color="magenta" dimColor={dimmed}>
        {name}
      </Text>
      <Text color="#C084FC" dimColor={dimmed}>
        {"↳ " + branch}
      </Text>
    </Box>
  );
}

export function WorktreeGrid({
  worktrees,
  cursor,
  termW,
  showSelection,
  marqueePhase,
  activeProject,
  dimmed,
}: Props) {
  if (!activeProject) {
    return (
      <Box borderStyle="round" borderColor="gray" paddingX={2} paddingY={1}>
        <Text dimColor>
          Sin proyecto activo. Pulsa p y luego a para añadir un repositorio.
        </Text>
      </Box>
    );
  }

  if (worktrees.length === 0) {
    return (
      <Box borderStyle="round" borderColor="gray" paddingX={2} paddingY={1}>
        <Text dimColor> (sin worktrees)</Text>
      </Box>
    );
  }

  const cols = gridCols(termW);
  const rows: React.ReactNode[] = [];

  for (let start = 0; start < worktrees.length; start += cols) {
    const cells: React.ReactNode[] = [];
    for (let c = 0; c < cols; c++) {
      const idx = start + c;
      if (idx >= worktrees.length) {
        cells.push(
          <Box
            key={`empty-${start}-${c}`}
            width={cardOuterW}
            minWidth={cardOuterW}
          />
        );
      } else {
        const wt = worktrees[idx]!;
        const sel = showSelection && idx === cursor;
        cells.push(
          <WorktreeCard
            key={wt.path}
            wt={wt}
            selected={sel}
            marqueePhase={marqueePhase}
            dimmed={dimmed}
          />
        );
      }
      if (c < cols - 1) {
        cells.push(<Box key={`gap-${start}-${c}`} width={gridColGap} />);
      }
    }
    rows.push(
      <Box key={`row-${start}`} flexDirection="row" marginBottom={1}>
        {cells}
      </Box>
    );
  }

  const panelW = gridTotalWidth(cols);
  const outer = Math.max(1, (termW ?? 80) - 4);
  return (
    <Box flexDirection="column" width={Math.min(panelW, outer)}>
      {rows}
    </Box>
  );
}
