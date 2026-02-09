# 前端API调用封装

## 一、Axios配置

### 1.1 Axios实例创建

```typescript
// src/api/client.ts
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import { message } from 'antd';

// API响应基础类型
interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}

// 创建Axios实例
const instance: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
instance.interceptors.request.use(
  (config) => {
    // 从localStorage获取token
    const token = localStorage.getItem('auth-storage');
    if (token) {
      try {
        const auth = JSON.parse(token);
        if (auth.state?.token) {
          config.headers.Authorization = `Bearer ${auth.state.token}`;
        }
      } catch (e) {
        console.error('Failed to parse auth storage:', e);
      }
    }

    // 添加请求ID
    config.headers['X-Request-ID'] = generateRequestId();

    // 添加时间戳
    config.headers['X-Request-Time'] = Date.now().toString();

    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
instance.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { data } = response;

    // 成功响应
    if (data.code === 0) {
      return response;
    }

    // 业务错误处理
    handleBusinessError(data);
    return Promise.reject(new Error(data.message || '请求失败'));
  },
  (error) => {
    // HTTP错误处理
    handleHttpError(error);
    return Promise.reject(error);
  }
);

// 处理业务错误
function handleBusinessError(data: ApiResponse) {
  switch (data.code) {
    case 1001:
      message.error('请求参数无效');
      break;
    case 1002:
      message.error('未授权，请重新登录');
      // 跳转到登录页
      window.location.href = '/auth/login';
      break;
    case 1003:
      message.error('禁止访问');
      break;
    case 1004:
      message.error('资源不存在');
      break;
    case 2001:
      message.error('用户不存在');
      break;
    case 2002:
      message.error('用户已存在');
      break;
    case 2003:
      message.error('密码错误');
      break;
    default:
      message.error(data.message || '操作失败');
  }
}

// 处理HTTP错误
function handleHttpError(error: any) {
  if (error.response) {
    const { status, data } = error.response;

    switch (status) {
      case 401:
        message.error('登录已过期，请重新登录');
        // 清除token并跳转登录
        localStorage.removeItem('auth-storage');
        window.location.href = '/auth/login';
        break;
      case 403:
        message.error('没有权限访问');
        break;
      case 404:
        message.error('请求的资源不存在');
        break;
      case 500:
        message.error('服务器错误，请稍后重试');
        break;
      case 502:
      case 503:
      case 504:
        message.error('服务暂时不可用，请稍后重试');
        break;
      default:
        message.error(data?.message || `请求失败 (${status})`);
    }
  } else if (error.request) {
    // 请求已发送但没有收到响应
    message.error('网络错误，请检查网络连接');
  } else {
    // 请求配置错误
    message.error('请求配置错误');
  }
}

// 生成请求ID
function generateRequestId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

export default instance;
```

### 1.2 类型定义

```typescript
// src/types/api.ts

// 通用分页请求参数
export interface PageParams {
  page?: number;
  pageSize?: number;
  orderBy?: string;
  order?: 'asc' | 'desc';
}

// 通用分页响应
export interface PageResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// 通用API响应
export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}
```

## 二、API模块封装

### 2.1 认证API

```typescript
// src/api/modules/auth.ts
import client from '../client';
import type { LoginRequest, LoginResponse, RefreshTokenRequest, RefreshTokenResponse } from '@/types/models';

// 登录
export const authApi = {
  // 用户登录
  login: async (data: LoginRequest) => {
    const response = await client.post<ApiResponse<LoginResponse>>('/auth/login', data);
    return response.data;
  },

  // 登出
  logout: async () => {
    const response = await client.post<ApiResponse<null>>('/auth/logout');
    return response.data;
  },

  // 刷新Token
  refreshToken: async (refreshToken: string) => {
    const response = await client.post<ApiResponse<RefreshTokenResponse>>('/auth/refresh', {
      refresh_token: refreshToken,
    });
    return response.data;
  },

  // 获取当前用户信息
  getCurrentUser: async () => {
    const response = await client.get<ApiResponse<User>>('/auth/me');
    return response.data;
  },
};
```

### 2.2 用户API

```typescript
// src/api/modules/user.ts
import client from '../client';
import type { User, CreateUserData, UpdateUserData, UserQueryParams, PageResponse } from '@/types/models';

export const userApi = {
  // 获取用户列表
  getUsers: async (params?: UserQueryParams) => {
    const response = await client.get<ApiResponse<PageResponse<User>>>('/users', { params });
    return response.data;
  },

  // 获取用户详情
  getUser: async (id: number) => {
    const response = await client.get<ApiResponse<User>>(`/users/${id}`);
    return response.data;
  },

  // 创建用户
  createUser: async (data: CreateUserData) => {
    const response = await client.post<ApiResponse<User>>('/users', data);
    return response.data;
  },

  // 更新用户
  updateUser: async (id: number, data: UpdateUserData) => {
    const response = await client.put<ApiResponse<User>>(`/users/${id}`, data);
    return response.data;
  },

  // 删除用户
  deleteUser: async (id: number) => {
    const response = await client.delete<ApiResponse<null>>(`/users/${id}`);
    return response.data;
  },

  // 重置密码
  resetPassword: async (id: number, newPassword: string) => {
    const response = await client.post<ApiResponse<null>>(`/users/${id}/reset-password`, {
      password: newPassword,
    });
    return response.data;
  },

  // 修改用户状态
  updateUserStatus: async (id: number, status: UserStatus) => {
    const response = await client.put<ApiResponse<User>>(`/users/${id}/status`, { status });
    return response.data;
  },
};
```

### 2.3 任务API

```typescript
// src/api/modules/task.ts
import client from '../client';
import type { Task, CreateTaskData, UpdateTaskData, TaskQueryParams, PageResponse } from '@/types/models';

export const taskApi = {
  // 获取任务列表
  getTasks: async (params?: TaskQueryParams) => {
    const response = await client.get<ApiResponse<PageResponse<Task>>>('/recordings', { params });
    return response.data;
  },

  // 获取任务详情
  getTask: async (id: number) => {
    const response = await client.get<ApiResponse<Task>>(`/recordings/${id}`);
    return response.data;
  },

  // 创建任务
  createTask: async (data: CreateTaskData) => {
    const response = await client.post<ApiResponse<Task>>('/recordings', data);
    return response.data;
  },

  // 更新任务
  updateTask: async (id: number, data: UpdateTaskData) => {
    const response = await client.put<ApiResponse<Task>>(`/recordings/${id}`, data);
    return response.data;
  },

  // 删除任务
  deleteTask: async (id: number) => {
    const response = await client.delete<ApiResponse<null>>(`/recordings/${id}`);
    return response.data;
  },

  // 启动任务
  startTask: async (id: number) => {
    const response = await client.post<ApiResponse<Task>>(`/recordings/${id}/start`);
    return response.data;
  },

  // 停止任务
  stopTask: async (id: number) => {
    const response = await client.post<ApiResponse<Task>>(`/recordings/${id}/stop`);
    return response.data;
  },

  // 取消任务
  cancelTask: async (id: number) => {
    const response = await client.post<ApiResponse<Task>>(`/recordings/${id}/cancel`);
    return response.data;
  },

  // 重试任务
  retryTask: async (id: number) => {
    const response = await client.post<ApiResponse<Task>>(`/recordings/${id}/retry`);
    return response.data;
  },

  // 获取任务日志
  getTaskLogs: async (id: number, page = 1, pageSize = 50) => {
    const response = await client.get<ApiResponse<PageResponse<TaskLog>>>(`/recordings/${id}/logs`, {
      params: { page, page_size: pageSize },
    });
    return response.data;
  },
};
```

### 2.4 会议API

```typescript
// src/api/modules/conference.ts
import client from '../client';
import type { Conference, CreateConferenceData, UpdateConferenceData, ConferenceQueryParams, PageResponse } from '@/types/models';

export const conferenceApi = {
  // 获取会议列表
  getConferences: async (params?: ConferenceQueryParams) => {
    const response = await client.get<ApiResponse<PageResponse<Conference>>>('/conferences', { params });
    return response.data;
  },

  // 获取会议详情
  getConference: async (id: number) => {
    const response = await client.get<ApiResponse<Conference>>(`/conferences/${id}`);
    return response.data;
  },

  // 创建会议
  createConference: async (data: CreateConferenceData) => {
    const response = await client.post<ApiResponse<Conference>>('/conferences', data);
    return response.data;
  },

  // 更新会议
  updateConference: async (id: number, data: UpdateConferenceData) => {
    const response = await client.put<ApiResponse<Conference>>(`/conferences/${id}`, data);
    return response.data;
  },

  // 删除会议
  deleteConference: async (id: number) => {
    const response = await client.delete<ApiResponse<null>>(`/conferences/${id}`);
    return response.data;
  },

  // 启动会议
  startConference: async (id: number) => {
    const response = await client.post<ApiResponse<Conference>>(`/conferences/${id}/start`);
    return response.data;
  },

  // 停止会议
  stopConference: async (id: number) => {
    const response = await client.post<ApiResponse<Conference>>(`/conferences/${id}/stop`);
    return response.data;
  },
};
```

### 2.5 文件API

```typescript
// src/api/modules/file.ts
import client from '../client';

export const fileApi = {
  // 上传文件
  upload: async (file: File, folder = 'uploads', isPublic = false, expiresIn = 0) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('folder', folder);
    formData.append('is_public', String(isPublic));
    if (expiresIn > 0) {
      formData.append('expires_in', String(expiresIn));
    }

    const response = await client.post<ApiResponse<{
      file_id: number;
      file_name: string;
      file_path: string;
      file_size: number;
      access_url: string;
    }>>('/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },

  // 上传多个文件
  uploadMultiple: async (files: File[], folder = 'uploads') => {
    const formData = new FormData();
    files.forEach(file => {
      formData.append('files', file);
    });
    formData.append('folder', folder);

    const response = await client.post<ApiResponse<any[]>>('/files/upload/multiple', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },

  // 获取文件列表
  getFiles: async (params?: {
    page?: number;
    pageSize?: number;
    fileType?: string;
  }) => {
    const response = await client.get<ApiResponse<PageResponse<UploadedFile>>>>('/files', { params });
    return response.data;
  },

  // 下载文件
  download: async (fileId: number) => {
    const response = await client.get(`/files/${fileId}/download`, {
      responseType: 'blob',
    });
    return response;
  },

  // 删除文件
  deleteFile: async (fileId: number) => {
    const response = await client.delete<ApiResponse<null>>(`/files/${fileId}`);
    return response.data;
  },

  // 生成分享链接
  shareFile: async (fileId: number, expiresIn: number, password?: string) => {
    const response = await client.post<ApiResponse<{ share_url: string }>>(`/files/${fileId}/share`, {
      expires_in: expiresIn,
      password,
    });
    return response.data;
  },

  // 获取用户配额
  getQuota: async () => {
    const response = await client.get<ApiResponse<UserStorageQuota>>('/files/quota');
    return response.data;
  },
};
```

### 2.6 通知API

```typescript
// src/api/modules/notification.ts
import client from '../client';
import type { Notification, NotificationQueryParams, PageResponse } from '@/types/models';

export const notificationApi = {
  // 获取通知列表
  getNotifications: async (params?: NotificationQueryParams) => {
    const response = await client.get<ApiResponse<PageResponse<Notification>>>('/notifications', { params });
    return response.data;
  },

  // 标记为已读
  markAsRead: async (id: number) => {
    const response = await client.put<ApiResponse<null>>(`/notifications/${id}/read`);
    return response.data;
  },

  // 全部标记为已读
  markAllAsRead: async () => {
    const response = await client.put<ApiResponse<null>>('/notifications/read-all');
    return response.data;
  },

  // 获取未读数量
  getUnreadCount: async () => {
    const response = await client.get<ApiResponse<{ count: number }>>('/notifications/unread-count');
    return response.data;
  },

  // 获取用户通知配置
  getSettings: async () => {
    const response = await client.get<ApiResponse<UserNotificationSetting>>('/notifications/settings');
    return response.data;
  },

  // 更新用户通知配置
  updateSettings: async (settings: Partial<UserNotificationSetting>) => {
    const response = await client.put<ApiResponse<UserNotificationSetting>>('/notifications/settings', settings);
    return response.data;
  },
};
```

## 三、请求封装Hooks

### 3.1 通用请求Hook

```typescript
// src/hooks/useRequest.ts
import { useState, useCallback } from 'react';
import { message } from 'antd';

interface UseRequestOptions<T> {
  manual?: boolean;           // 手动触发
  onSuccess?: (data: T) => void;
  onError?: (error: Error) => void;
  onFinally?: () => void;
}

export function useRequest<T>(
  requestFn: () => Promise<T>,
  options: UseRequestOptions<T> = {}
) {
  const { manual = false, onSuccess, onError, onFinally } = options;

  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const execute = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await requestFn();
      setData(result);
      onSuccess?.(result);
      return result;
    } catch (err: any) {
      const error = err instanceof Error ? err : new Error(err.message || '请求失败');
      setError(error);
      onError?.(error);
      throw error;
    } finally {
      setLoading(false);
      onFinally?.();
    }
  }, [requestFn, onSuccess, onError, onFinally]);

  // 自动执行
  if (!manual) {
    execute();
  }

  return {
    loading,
    data,
    error,
    execute,
    mutate: execute, // 别名
  };
}

// 使用示例
function TaskCreate() {
  const navigate = useNavigate();
  const [form] = Form.useForm();

  const { loading, execute: createTask } = useRequest(
    () => taskApi.createTask(form.getFieldsValue()),
    {
      onSuccess: (data) => {
        message.success('任务创建成功');
        navigate(`/tasks/${data.id}`);
      },
      onError: (error) => {
        message.error(error.message || '创建失败');
      },
    }
  );

  const handleSubmit = () => {
    form.validateFields().then(createTask);
  };

  return <Form onFinish={handleSubmit}>...</Form>;
}
```

### 3.2 分页请求Hook

```typescript
// src/hooks/usePaginatedRequest.ts
import { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';

interface Pagination {
  page: number;
  pageSize: number;
  total: number;
}

export function usePaginatedRequest<T>(
  queryKey: any[],
  requestFn: (params: PageParams) => Promise<PageResponse<T>>,
  initialPageSize = 20
) {
  const [pagination, setPagination] = useState<Pagination>({
    page: 1,
    pageSize: initialPageSize,
    total: 0,
  });

  const { isLoading, data, refetch } = useQuery({
    queryKey: [...queryKey, pagination.page, pagination.pageSize],
    queryFn: () => requestFn({
      page: pagination.page,
      pageSize: pagination.pageSize,
    }),
    onSuccess: (response) => {
      setPagination(prev => ({
        ...prev,
        total: response.total,
      }));
    },
  });

  const handlePageChange = (page: number, pageSize: number) => {
    setPagination({
      page,
      pageSize,
      total: pagination.total,
    });
  };

  return {
    loading: isLoading,
    data: data?.items || [],
    total: pagination.total,
    page: pagination.page,
    pageSize: pagination.pageSize,
    refetch,
    onPageChange: handlePageChange,
  };
}
```

## 四、文件上传封装

### 4.1 上传组件Hook

```typescript
// src/hooks/useUpload.ts
import { useState, useCallback } from 'react';
import { UploadFile, UploadProps } from 'antd';
import { message } from 'antd';
import { fileApi } from '@/api/modules/file';

export function useUpload(options?: {
  folder?: string;
  isPublic?: boolean;
  expiresIn?: number;
  maxSize?: number; // 字节
  accept?: string[];
}) {
  const {
    folder = 'uploads',
    isPublic = false,
    expiresIn = 0,
    maxSize = 100 * 1024 * 1024, // 100MB
    accept = [],
  } = options || {};

  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploading, setUploading] = useState(false);

  const handleChange: UploadProps['onChange'] = ({ fileList: newFileList }) => {
    setFileList(newFileList);
  };

  const handleUpload = async () => {
    const file = fileList[0]?.originFileObj;
    if (!file) {
      message.error('请选择文件');
      return;
    }

    setUploading(true);
    try {
      const result = await fileApi.upload(file, folder, isPublic, expiresIn);

      message.success('上传成功');
      setFileList([]);
      return result;
    } catch (error: any) {
      message.error(error.message || '上传失败');
      throw error;
    } finally {
      setUploading(false);
    }
  };

  const beforeUpload = (file: File) => {
    // 检查文件大小
    if (file.size > maxSize) {
      message.error(`文件大小不能超过 ${maxSize / 1024 / 1024}MB`);
      return Upload.LIST_IGNORE;
    }

    // 检查文件类型
    if (accept.length > 0) {
      const fileExt = file.name.split('.').pop();
      if (!accept.includes(`.${fileExt}`)) {
        message.error('不支持的文件类型');
        return Upload.LIST_IGNORE;
      }
    }

    return true;
  };

  const uploadProps: UploadProps = {
    listType: 'picture',
    fileList,
    beforeUpload,
    onChange: handleChange,
    onRemove: () => {
      setFileList([]);
      return true;
    },
  };

  return {
    uploadProps,
    uploading,
    handleUpload,
    fileList,
  };
}
```

### 4.2 拖拽上传Hook

```typescript
// src/hooks/useDragUpload.ts
import { useState } from 'react';
import { Upload, message } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import type { UploadProps } from 'antd';

export function useDragUpload(options?: {
  multiple?: boolean;
  maxSize?: number;
  accept?: string[];
  onUpload: (files: File[]) => Promise<void>;
}) {
  const {
    multiple = false,
    maxSize = 100 * 1024 * 1024,
    accept = [],
    onUpload,
  } = options || {};

  const [uploading, setUploading] = useState(false);

  const handleDrop: UploadProps['customRequest'] = async (options) => {
    const { file } = options;
    setUploading(true);

    try {
      await onUpload([file as File]);
      options.onSuccess!(file);
    } catch (error) {
      options.onError!(error as Error);
    } finally {
      setUploading(false);
    }
  };

  const beforeUpload = (file: File) => {
    if (file.size > maxSize) {
      message.error(`文件大小不能超过 ${maxSize / 1024 / 1024}MB`);
      return false;
    }

    if (accept.length > 0) {
      const fileExt = file.name.split('.').pop();
      if (!accept.includes(`.${fileExt}`)) {
        message.error('不支持的文件类型');
        return false;
      }
    }

    return true;
  };

  const uploadProps: UploadProps = {
    name: 'file',
    multiple,
    customRequest: handleDrop,
    beforeUpload,
    showUploadList: false,
  };

  const draggerProps = {
    name: 'file',
    multiple,
    customRequest: handleDrop,
    beforeUpload,
    showUploadList: false,
  };

  return {
    uploadProps,
    draggerProps,
    uploading,
  };
}
```

## 五、WebSocket封装

### 5.1 WebSocket Hook

```typescript
// src/hooks/useWebSocket.ts
import { useEffect, useRef, useCallback } from 'react';
import { eventBus } from '@/utils/eventBus';

export function useWebSocket(url: string, options?: {
  onMessage?: (event: MessageEvent) => void;
  onError?: (event: Event) => void;
  onClose?: (event: CloseEvent) => void;
  onOpen?: (event: Event) => void;
  reconnect?: boolean;
  reconnectInterval?: number;
}) {
  const {
    onMessage,
    onError,
    onClose,
    onOpen,
    reconnect = true,
    reconnectInterval = 3000,
  } = options || {};

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<NodeJS.Timeout>();

  const connect = useCallback(() => {
    const token = localStorage.getItem('auth-storage');
    let wsUrl = url;

    // 添加token到URL
    if (token) {
      try {
        const auth = JSON.parse(token);
        if (auth.state?.token) {
          wsUrl += `?token=${auth.state.token}`;
        }
      } catch (e) {}
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = (event) => {
      console.log('WebSocket connected');
      onOpen?.(event);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // 发送到事件总线
        eventBus.emit(data.type, data);
      } catch {
        // 非JSON消息直接传递
        eventBus.emit('message', event.data);
      }
      onMessage?.(event);
    };

    ws.onerror = (event) => {
      console.error('WebSocket error:', event);
      onError?.(event);
    };

    ws.onclose = (event) => {
      console.log('WebSocket closed');
      onClose?.(event);

      // 自动重连
      if (reconnect) {
        reconnectTimerRef.current = setTimeout(() => {
          connect();
        }, reconnectInterval);
      }
    };
  }, [url, onMessage, onError, onClose, onOpen, reconnect, reconnectInterval]);

  useEffect(() => {
    connect();

    return () => {
      // 清理
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
    };
  }, [connect]);

  // 发送消息
  const send = useCallback((data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn('WebSocket is not connected');
    }
  }, []);

  return {
    send,
    ws: wsRef.current,
  };
}

// 使用示例
function TaskMonitor() {
  const { send } = useWebSocket(`${WS_URL}/tasks`, {
    onMessage: (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'task_status_changed') {
        // 处理任务状态变化
        console.log('Task status changed:', data);
      }
    },
  });

  return <div>Task Monitor</div>;
}
```

## 六、相关文档

- [31-前端架构设计.md](./31-前端架构设计.md)
- [32-前端状态管理.md](./32-前端状态管理.md)
- [33-前端路由与权限.md](./33-前端路由与权限.md)
- [35-前端认证流程.md](./35-前端认证流程.md)
