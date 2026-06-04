import React, { useEffect, useState } from 'react';
import { API } from 'utils/api';
import { showError } from 'utils/common';
import { marked } from 'marked';
import { Box, Container, Typography } from '@mui/material';
import MainCard from 'ui-component/cards/MainCard';

const About = () => {
  const [about, setAbout] = useState('');
  const [aboutLoaded, setAboutLoaded] = useState(false);

  const displayAbout = async () => {
    setAbout(localStorage.getItem('about') || '');
    const res = await API.get('/api/about');
    const { success, message, data } = res.data;
    if (success) {
      let aboutContent = data;
      if (!data.startsWith('https://')) {
        aboutContent = marked.parse(data);
      }
      setAbout(aboutContent);
      localStorage.setItem('about', aboutContent);
    } else {
      showError(message);
      setAbout('加载关于内容失败...');
    }
    setAboutLoaded(true);
  };

  useEffect(() => {
    displayAbout().then();
  }, []);

  return (
    <>
      {aboutLoaded && about === '' ? (
        <>
          <Box>
            <Container sx={{ paddingTop: '40px' }}>
              <MainCard title="关于">
                <Typography variant="body2">
                  <strong>One API · 二次开发版 v1.0.0</strong>
                  <br />
                  本版本在原版基础上增加了以下功能：
                  <br />
                  • 新增 NVIDIA NIM、Perplexity、Cerebras、GitHub Models 渠道支持
                  <br />
                  • 渠道编辑界面一键获取可用模型
                  <br />
                  • 模型倍率从 OpenRouter 同步（设置页面配置）
                  <br />
                  • 从 OpenRouter 自动同步并创建渠道
                  <br />
                  • Berry 主题默认启用，暗色模式参考 VSCode Dark++ 配色
                  <br />
                  <br />
                  原项目仓库地址：
                  <a href="https://github.com/songquanpeng/one-api">https://github.com/songquanpeng/one-api</a>
                  <br />
                  本分支仓库地址：
                  <a href="https://github.com/w-run/one-api">https://github.com/w-run/one-api</a>
                  <br />
                  <br />
                  二次开发维护：<strong>W/Run</strong>
                </Typography>
              </MainCard>
            </Container>
          </Box>
        </>
      ) : (
        <>
          <Box>
            {about.startsWith('https://') ? (
              <iframe title="about" src={about} style={{ width: '100%', height: '100vh', border: 'none' }} />
            ) : (
              <>
                <Container>
                  <div style={{ fontSize: 'larger' }} dangerouslySetInnerHTML={{ __html: about }}></div>
                </Container>
              </>
            )}
          </Box>
        </>
      )}
    </>
  );
};

export default About;
