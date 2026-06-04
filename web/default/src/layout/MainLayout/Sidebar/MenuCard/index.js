import { useSelector } from 'react-redux';

// material-ui
import { styled, useTheme } from '@mui/material/styles';
import {
  Card,
  CardContent,
  Typography
} from '@mui/material';
import { useNavigate } from 'react-router-dom';

const CardStyle = styled(Card)(({ theme }) => ({
  background: theme.typography.menuChip.background,
  marginBottom: '22px',
  overflow: 'hidden',
  position: 'relative',
  '&:after': {
    content: '""',
    position: 'absolute',
    width: '157px',
    height: '157px',
    background: theme.palette.primary[200],
    borderRadius: '50%',
    top: '-105px',
    right: '-96px'
  }
}));

const MenuCard = () => {
  const theme = useTheme();
  const account = useSelector((state) => state.account);
  const navigate = useNavigate();

  return (
    <CardStyle onClick={() => navigate('/panel/profile')} sx={{ cursor: 'pointer' }}>
      <CardContent sx={{ padding: '16px 24px' }}>
        <Typography 
          variant="subtitle1" 
          sx={{ 
            color: theme.palette.primary[800],
            fontSize: '20px',
            fontWeight: 'bold'
          }}
        >
          {account.user?.username || '用户'}
        </Typography>
        <Typography variant="caption"> 欢迎回来~ </Typography>
      </CardContent>
    </CardStyle>
  );
};

export default MenuCard;