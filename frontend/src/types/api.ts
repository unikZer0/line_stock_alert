export interface ApiResponse<T> {
  success: true;
  data: T;
  message?: string;
}

export interface ApiPageResponse<T> extends ApiResponse<T> {
  meta: {
    page: number;
    limit: number;
    total: number;
  };
}

export interface ApiErrorResponse {
  success: false;
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
}
