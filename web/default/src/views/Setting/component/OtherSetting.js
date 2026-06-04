import { useState, useEffect } from 'react';
import SubCard from 'ui-component/cards/SubCard';
import {
    Stack,
    FormControl,
    InputLabel,
    OutlinedInput,
    Button,
    Alert,
    TextField
} from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { showError, showSuccess } from 'utils/common'; //,
import { API } from 'utils/api';

const OtherSetting = () => {
  let [inputs, setInputs] = useState({
    Footer: '',
    Notice: '',
    About: '',
    SystemName: '',
    Logo: '',
    HomePageContent: '',
    Theme: '',
    ModelRatioSyncURL: '',
  });
  let [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      setInputs(newInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions().then();
  }, []);

  const updateOption = async (key, value) => {
    setLoading(true);
    const res = await API.put('/api/option/', {
      key,
      value
    });
    const { success, message } = res.data;
    if (success) {
      setInputs((inputs) => ({ ...inputs, [key]: value }));
      showSuccess('保存成功');
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleInputChange = async (event) => {
    let { name, value } = event.target;
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const submitNotice = async () => {
    await updateOption('Notice', inputs.Notice);
  };

  const submitFooter = async () => {
    await updateOption('Footer', inputs.Footer);
  };

  const submitSystemName = async () => {
    await updateOption('SystemName', inputs.SystemName);
  };

  const submitTheme = async () => {
    await updateOption('Theme', inputs.Theme);
  };

  const submitLogo = async () => {
    await updateOption('Logo', inputs.Logo);
  };

  const submitAbout = async () => {
    await updateOption('About', inputs.About);
  };

  const submitOption = async (key) => {
    await updateOption(key, inputs[key]);
  };

  const syncRatios = async () => {
    setSyncing(true);
    try {
      const res = await API.post('/api/option/sync_ratios', {
        url: inputs.ModelRatioSyncURL
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(`同步成功，共获取 ${data.count} 个模型的倍率`);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setSyncing(false);
    }
  };

  const syncRatiosOpenRouter = async () => {
    setSyncing(true);
    try {
      const res = await API.post('/api/option/sync_ratios', {
        url: 'https://openrouter.ai/api/v1/models'
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(`同步成功，共获取 ${data.count} 个模型的倍率`);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setSyncing(false);
    }
  };

  return (
    <>
      <Stack spacing={2}>
        <SubCard title="通用设置">
          <Grid container spacing={{ xs: 3, sm: 2, md: 4 }}>
            <Grid xs={12}>
              <FormControl fullWidth>
                <TextField
                  multiline
                  maxRows={15}
                  id="Notice"
                  label="公告"
                  value={inputs.Notice}
                  name="Notice"
                  onChange={handleInputChange}
                  minRows={10}
                  placeholder="在此输入新的公告内容，支持 Markdown & HTML 代码"
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitNotice}>
                保存公告
              </Button>
            </Grid>
          </Grid>
        </SubCard>
        <SubCard title="个性化设置">
          <Grid container spacing={{ xs: 3, sm: 2, md: 4 }}>
            <Grid xs={12}>
              <FormControl fullWidth>
                <InputLabel htmlFor="SystemName">系统名称</InputLabel>
                <OutlinedInput
                  id="SystemName"
                  name="SystemName"
                  value={inputs.SystemName || ''}
                  onChange={handleInputChange}
                  label="系统名称"
                  placeholder="在此输入系统名称"
                  disabled={loading}
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitSystemName}>
                设置系统名称
              </Button>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <InputLabel htmlFor="Theme">主题名称</InputLabel>
                <OutlinedInput
                    id="Theme"
                    name="Theme"
                    value={inputs.Theme || ''}
                    onChange={handleInputChange}
                    label="主题名称"
                    placeholder="请输入主题名称"
                    disabled={loading}
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitTheme}>
                设置主题（重启生效）
              </Button>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <InputLabel htmlFor="Logo">Logo 图片地址</InputLabel>
                <OutlinedInput
                  id="Logo"
                  name="Logo"
                  value={inputs.Logo || ''}
                  onChange={handleInputChange}
                  label="Logo 图片地址"
                  placeholder="在此输入Logo 图片地址"
                  disabled={loading}
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitLogo}>
                设置 Logo
              </Button>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <TextField
                  multiline
                  maxRows={15}
                  id="HomePageContent"
                  label="首页内容"
                  value={inputs.HomePageContent}
                  name="HomePageContent"
                  onChange={handleInputChange}
                  minRows={10}
                  placeholder="在此输入首页内容，支持 Markdown & HTML 代码，设置后首页的状态信息将不再显示。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为首页。"
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={() => submitOption('HomePageContent')}>
                保存首页内容
              </Button>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <TextField
                  multiline
                  maxRows={15}
                  id="About"
                  label="关于"
                  value={inputs.About}
                  name="About"
                  onChange={handleInputChange}
                  minRows={10}
                  placeholder="在此输入新的关于内容，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为关于页面。"
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitAbout}>
                保存关于
              </Button>
            </Grid>
            <Grid xs={12}>
              <Alert severity="warning">
                移除 One API 的版权标识必须首先获得授权，项目维护需要花费大量精力，如果本项目对你有意义，请主动支持本项目。
              </Alert>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <TextField
                  multiline
                  maxRows={15}
                  id="Footer"
                  label="页脚"
                  value={inputs.Footer}
                  name="Footer"
                  onChange={handleInputChange}
                  minRows={10}
                  placeholder="在此输入新的页脚，留空则使用默认页脚，支持 HTML 代码"
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Button variant="contained" onClick={submitFooter}>
                设置页脚
              </Button>
            </Grid>
          </Grid>
        </SubCard>
        <SubCard title="倍率同步">
          <Grid container spacing={{ xs: 3, sm: 2, md: 4 }}>
            <Grid xs={12}>
              <Alert severity="info">
                从外部 API 自动同步模型倍率，支持 OpenRouter 标准格式。配置后，点击同步按钮即可将外部定价自动转换为系统倍率。
              </Alert>
            </Grid>
            <Grid xs={12}>
              <Stack direction="row" spacing={2}>
                <Button variant="contained" disabled={syncing} onClick={syncRatiosOpenRouter}>
                  {syncing ? '同步中...' : '从 OpenRouter 同步'}
                </Button>
              </Stack>
            </Grid>
            <Grid xs={12}>
              <FormControl fullWidth>
                <InputLabel htmlFor="ModelRatioSyncURL">自定义倍率同步地址</InputLabel>
                <OutlinedInput
                  id="ModelRatioSyncURL"
                  name="ModelRatioSyncURL"
                  value={inputs.ModelRatioSyncURL || ''}
                  onChange={handleInputChange}
                  label="自定义倍率同步地址"
                  placeholder="https://example.com/api/models"
                  disabled={loading}
                />
              </FormControl>
            </Grid>
            <Grid xs={12}>
              <Stack direction="row" spacing={2}>
                <Button variant="contained" onClick={() => updateOption('ModelRatioSyncURL', inputs.ModelRatioSyncURL)} disabled={loading}>
                  保存地址
                </Button>
                <Button variant="outlined" disabled={syncing} onClick={syncRatios}>
                  {syncing ? '同步中...' : '从自定义地址同步'}
                </Button>
              </Stack>
            </Grid>
          </Grid>
        </SubCard>
      </Stack>
    </>
  );
};

export default OtherSetting;
