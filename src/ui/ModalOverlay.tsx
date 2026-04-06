import React from "react";
import { Box } from "ink";

type Props = {
  width: number;
  height: number;
  backdrop: React.ReactNode;
  modal: React.ReactNode;
};

/** Centers modal above backdrop (Bubble Tea `overlayModalCenter` parity). */
export function ModalOverlay({
  width,
  height,
  backdrop,
  modal,
}: Props) {
  return (
    <Box flexDirection="column" width={width} minHeight={height}>
      <Box flexDirection="column" width={width}>
        {backdrop}
      </Box>
      <Box
        position="absolute"
        flexDirection="column"
        width={width}
        height={height}
        alignItems="center"
        justifyContent="center"
      >
        {modal}
      </Box>
    </Box>
  );
}
