import Card from '@mui/material/Card';
import { Box } from '@mui/material';

import React from 'react';

export default function UserCard({ children }) {
  return (
    <Card>
      <Box
        sx={{
          p: (theme) => theme.spacing(4, 3, 3, 3)
        }}
      >
        {children}
      </Box>
    </Card>
  );
}