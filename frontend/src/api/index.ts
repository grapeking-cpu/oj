import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error.response?.data || error)
  }
)

// ========== 用户相关 ==========

export interface LoginParams {
  username: string
  password: string
}

export interface RegisterParams extends LoginParams {
  email: string
  captcha_key: string
  captcha_code: string
}

export const login = (data: LoginParams) =>
  api.post('/user/login', data)

export const register = (data: RegisterParams) =>
  api.post('/user/register', data)

export const getCaptcha = () =>
  api.get('/user/captcha')

export const getUserInfo = () =>
  api.get('/user/info')

export const updateProfile = (data: { nickname?: string; avatar?: string }) =>
  api.put('/user/profile', data)

export const changePassword = (data: { old_password: string; new_password: string }) =>
  api.put('/user/password', data)

export const logout = () =>
  api.post('/user/logout')

// ========== 题目相关 ==========

export interface Problem {
  id: number
  title: string
  difficulty: number
  tags: string[]
  accept_rate: number
}

export interface ProblemDetail extends Problem {
  description: string
  input_format: string
  output_format: string
  sample_io: { input: string; output: string }[]
  hint?: string
  time_limit: number
  memory_limit: number
  is_spj: boolean
}

export const getProblemList = (params?: { page?: number; page_size?: number; difficulty?: number; tags?: string; search?: string }) =>
  api.get<{ list: Problem[]; total: number }>('/problems', { params })

export const getProblem = (id: number) =>
  api.get<ProblemDetail>(`/problems/${id}`)

export interface SubmitParams {
  problem_id: number
  language_id: number
  code: string
}

export const createSubmit = (data: SubmitParams) =>
  api.post('/submit', data)

export const getSubmit = (submit_id: string) =>
  api.get(`/submit/${submit_id}`)

export const getMySubmits = (params?: { problem_id?: number; status?: string }) =>
  api.get('/my/submits', { params })

// ========== 语言相关 ==========

export interface Language {
  id: number
  name: string
  slug: string
  source_filename: string
  docker_image: string
  enabled: boolean
}

export const getLanguageList = () =>
  api.get<Language[]>('/languages')

// ========== 比赛相关 ==========

export interface Contest {
  id: number
  title: string
  type: string
  status: 'upcoming' | 'running' | 'ended'
  start_time: string
  end_time: string
  participant_count: number
}

export const getContestList = (params?: { type?: string; status?: string }) =>
  api.get<Contest[]>('/contests', { params })

export const getContest = (id: number) =>
  api.get(`/contests/${id}`)

export const getContestRank = (id: number, force?: boolean) =>
  api.get(`/contests/${id}/rank`, { params: { force: force ? 1 : 0 } })

export const joinContest = (id: number, password?: string) =>
  api.post(`/contests/${id}/join`, { password })

export const contestSubmit = (id: number, data: SubmitParams & { problem_letter: string }) =>
  api.post(`/contests/${id}/submit`, data)

export default api
