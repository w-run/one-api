// material-ui
import { Container, Box, Link } from '@mui/material';
import React from 'react';
import { useSelector } from 'react-redux';

// ==============================|| FOOTER ||============================== //

const Footer = () => {
  const siteInfo = useSelector((state) => state.siteInfo);
  const year = new Date().getFullYear();
  const fromYear = 2025;
  const yearRange = year > fromYear ? `${fromYear}-${year}` : `${fromYear}`;

  return (
    <Container sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '64px' }}>
      <Box sx={{ textAlign: 'center' }}>
        {siteInfo.footer_html ? (
          <div className="custom-footer" dangerouslySetInnerHTML={{ __html: siteInfo.footer_html }}></div>
        ) : (
          <span>
            ©{yearRange}{' '}
            <Link href="https://github.com/w-run" target="_blank" underline="hover">
              W/Run
            </Link>
            .
          </span>
        )}
      </Box>
    </Container>
  );
};

export default Footer;
