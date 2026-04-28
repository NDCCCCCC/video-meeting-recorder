import React, { useState, useEffect } from 'react';
import {
  Form,
  Input,
  Switch,
  Button,
  message,
  Modal,
  Alert,
  Tag,
  Space,
  Select,
  Tooltip,
} from 'antd';
import {
  WarningOutlined,
  CheckCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import {
  getAuthConfig,
  updateAuthConfig,
  testADConnection,
} from '@/api/auth';
import type { AuthConfigResponse, ADValidationResult } from '@/types/auth';

const { Option } = Select;

const AuthConfigPage: React.FC = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [config, setConfig] = useState<AuthConfigResponse | null>(null);
  const [validationResult, setValidationResult] = useState<ADValidationResult | null>(null);

  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    setLoading(true);
    try {
      const response = await getAuthConfig();
      setConfig(response.data);
      form.setFieldsValue(response.data);
    } catch (error) {
      message.error('获取配置失败');
    } finally {
      setLoading(false);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    setValidationResult(null);
    try {
      const values = await form.validateFields(['ad']);
      const response = await testADConnection(values.ad);

      setValidationResult(response.data);

      if (response.data.valid) {
        message.success('AD连接测试成功');
      } else {
        message.error('AD连接测试失败: ' + response.data.errors?.join(', '));
      }
    } catch (error: any) {
      message.error('连接测试失败: ' + (error.response?.data?.message || error.message));
    } finally {
      setTesting(false);
    }
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();

      // Show warning if using port 389 (per D-12, D-14)
      if (values.mode === 'ad' && !values.ad.use_tls && !validationResult?.valid) {
        Modal.confirm({
          title: '安全警告',
          icon: <WarningOutlined style={{ color: '#ff4d4f' }} />,
          content: '⚠️ 使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。是否继续？',
          okText: '继续保存',
          cancelText: '取消',
          okButtonProps: { danger: true },
          onOk: async () => {
            await saveConfig(values);
          },
        });
      } else {
        await saveConfig(values);
      }
    } catch (error: any) {
      message.error('保存配置失败');
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async (values: any) => {
    try {
      await updateAuthConfig(values);
      message.success('配置已更新');
      fetchConfig();
    } catch (error: any) {
      message.error(error.response?.data?.message || '保存失败');
    }
  };

  const handleModeSwitch = (newMode: 'local' | 'ad') => {
    if (newMode === 'ad' && !validationResult?.valid) {
      Modal.warning({
        title: '无法切换到AD模式',
        content: '请先测试AD连接并确保配置验证通过。',
      });
      return false;
    }

    if (newMode === 'ad') {
      Modal.confirm({
        title: '确认切换认证模式',
        content: '切换到AD模式后，所有用户将使用域控账号登录。请确认当前AD配置已验证通过。',
        onOk: () => {
          form.setFieldValue('mode', newMode);
        },
      });
      return false;
    }

    return true;
  };

  return (
    <div style={{ padding: '24px', maxWidth: '800px', margin: '0 auto' }}>
      <h1 style={{ fontSize: '28px', fontWeight: 600, marginBottom: '24px' }}>
        认证配置
      </h1>

      <Form
        form={form}
        layout="vertical"
        initialValues={{ mode: 'local', ad: { use_tls: true } }}
      >
        <Form.Item
          label="认证模式"
          name="mode"
          rules={[{ required: true, message: '请选择认证模式' }]}
        >
          <Select
            style={{ width: '100%' }}
            onChange={(value) => handleModeSwitch(value)}
          >
            <Option value="local">本地认证</Option>
            <Option value="ad">AD域控认证</Option>
          </Select>
        </Form.Item>

        {config?.mode && (
          <Alert
            message={
              <Space>
                当前模式: <Tag color={config.mode === 'local' ? 'blue' : 'green'}>
                  {config.mode === 'local' ? '本地认证' : 'AD域控认证'}
                </Tag>
              </Space>
            }
            type="info"
            style={{ marginBottom: '16px' }}
          />
        )}

        <Form.Item noStyle shouldUpdate={(prev, curr) => prev.mode !== curr.mode}>
          {({ getFieldValue }) =>
            getFieldValue('mode') === 'ad' ? (
              <>
                <Alert
                  message="AD域控配置"
                  description="配置AD服务器连接信息，端口636推荐用于生产环境"
                  type="info"
                  showIcon
                  style={{ marginBottom: '16px' }}
                />

                <Form.Item
                  label={
                    <Space>
                      AD服务器地址
                      {!getFieldValue(['ad', 'use_tls']) && (
                        <Tooltip title="使用LDAP 389端口时密码将以明文传输">
                          <WarningOutlined style={{ color: '#ff4d4f' }} />
                        </Tooltip>
                      )}
                    </Space>
                  }
                  name={['ad', 'server']}
                  rules={[{ required: true, message: '请输入AD服务器地址' }]}
                >
                  <Input placeholder="ad.example.com:636" />
                </Form.Item>

                <Form.Item
                  label="BindDN"
                  name={['ad', 'bind_dn']}
                  rules={[{ required: true, message: '请输入BindDN' }]}
                >
                  <Input placeholder="cn=admin,cn=users,dc=example,dc=com" />
                </Form.Item>

                <Form.Item
                  label="管理员密码"
                  name={['ad', 'password']}
                  rules={[{ required: true, message: '请输入管理员密码' }]}
                >
                  <Input.Password placeholder="••••••••" />
                </Form.Item>

                <Form.Item
                  label="BaseDN"
                  name={['ad', 'base_dn']}
                  rules={[{ required: true, message: '请输入BaseDN' }]}
                >
                  <Input placeholder="dc=example,dc=com" />
                </Form.Item>

                <Form.Item
                  label="启用LDAPS"
                  name={['ad', 'use_tls']}
                  valuePropName="checked"
                  tooltip="推荐使用LDAPS端口636以加密传输"
                >
                  <Switch />
                </Form.Item>

                {!getFieldValue(['ad', 'use_tls']) && (
                  <Alert
                    message="安全警告"
                    description="⚠️ 使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。"
                    type="warning"
                    showIcon
                    icon={<WarningOutlined />}
                    style={{ marginBottom: '16px' }}
                  />
                )}

                <Form.Item>
                  <Space>
                    <Button
                      onClick={handleTestConnection}
                      loading={testing}
                      icon={testing ? <LoadingOutlined /> : undefined}
                    >
                      测试连接
                    </Button>

                    {validationResult && (
                      <Space direction="vertical" style={{ width: '100%' }}>
                        {validationResult.valid ? (
                          <Alert
                            message={
                              <Space>
                                <CheckCircleOutlined />
                                连接测试成功
                                {validationResult.response_time &&
                                  ` (${validationResult.response_time}ms)`}
                              </Space>
                            }
                            type="success"
                          />
                        ) : (
                          <Alert
                            message="连接测试失败"
                            description={validationResult.errors?.join(', ')}
                            type="error"
                          />
                        )}

                        {validationResult.warnings &&
                          validationResult.warnings.length > 0 && (
                            <Alert
                              message="警告"
                              description={validationResult.warnings.join('\n')}
                              type="warning"
                            />
                          )}
                      </Space>
                    )}
                  </Space>
                </Form.Item>
              </>
            ) : null
          }
        </Form.Item>

        <Form.Item>
          <Button type="primary" onClick={handleSave} loading={loading}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
};

export default AuthConfigPage;
