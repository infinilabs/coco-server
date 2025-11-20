import { Button } from "antd";
import { ExclamationCircleOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

interface OAuthConfig {
  client_id?: string;
  client_secret?: string;
  auth_url?: string;
  redirect_url?: string;
  token_url?: string;
}

// Predefined validation presets for common OAuth providers
export const OAuthValidationPresets = {
  // Standard OAuth 2.0 (5 fields) - Google Drive, etc.
  standard: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
    { field: 'redirect_url', required: true, label: 'Redirect URL' },
    { field: 'token_url', required: true, label: 'Token URL' },
  ],
  
  // Minimal OAuth (3 fields) - Some providers might not need redirect_url/token_url
  minimal: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
  ],
  
  // Backend-only validation (1 field) - Just check auth_url to enable OAuth UI
  backendOnly: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
  ],
  
  // Credentials only (2 fields) - When endpoints are hardcoded in backend
  credentialsOnly: [
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
  ],
  
  // Authorization only (1 field) - Just need auth endpoint
  authOnly: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
  ],
  
  // Custom validation examples for specific providers
  googleDrive: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
    { field: 'redirect_url', required: true, label: 'Redirect URL' },
    { field: 'token_url', required: true, label: 'Token URL' },
  ],
  
  feishuLark: [
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
  ],
  
  // Example of conditional validation - require redirect_url only if not using default
  conditionalExample: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
    { 
      field: 'redirect_url', 
      required: (config: OAuthConfig) => {
        // Only require redirect_url if auth_url is provided (conditional logic)
        return !!config.auth_url;
      }, 
      label: 'Redirect URL' 
    },
    { field: 'token_url', required: true, label: 'Token URL' },
  ],
  
  // Example with custom validator - validate URL format
  withCustomValidation: [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
    { 
      field: 'redirect_url', 
      required: true, 
      label: 'Redirect URL',
      customValidator: (config: OAuthConfig) => {
        if (!config.redirect_url) return { valid: false, message: 'Redirect URL is required' };
        
        // Basic URL validation
        try {
          new URL(config.redirect_url);
          return { valid: true };
        } catch {
          return { valid: false, message: 'Redirect URL must be a valid URL' };
        }
      }
    },
  ],
} as const;

interface ValidationRule {
  field: keyof OAuthConfig;
  required: boolean | ((config: OAuthConfig) => boolean); // Can be conditional
  label?: string;
  customValidator?: (config: OAuthConfig) => { valid: boolean; message?: string }; // Custom validation logic
}

interface OAuthConnectProps {
  connector: {
    id?: string;
    config?: OAuthConfig;
    name?: string;
  };
  connectUrl?: string;
  missingConfigMessage?: string;
  connectButtonText?: string;
  className?: string;
  validationRules?: ValidationRule[]; // Configurable validation rules
}

export default function OAuthConnect({ 
  connector, 
  connectUrl,
  missingConfigMessage,
  connectButtonText,
  className = "flex items-center justify-between px-20px",
  validationRules
}: OAuthConnectProps) {
  const { t } = useTranslation();
  const nav = useNavigate();

  // Default validation rules - require all standard OAuth fields
  const defaultValidationRules: ValidationRule[] = [
    { field: 'auth_url', required: true, label: 'Authorization URL' },
    { field: 'client_id', required: true, label: 'Client ID' },
    { field: 'client_secret', required: true, label: 'Client Secret' },
    { field: 'redirect_url', required: true, label: 'Redirect URL' },
    { field: 'token_url', required: true, label: 'Token URL' },
  ];

  // Use provided validation rules or fall back to defaults
  const rules = validationRules || defaultValidationRules;

  const onConnectClick = () => {
    const config = connector?.config || {};

    // Check validation rules
    const missingFields: string[] = [];
    const missingFieldLabels: string[] = [];
    const customErrors: string[] = [];
    
    rules.forEach(rule => {
      // Check if field is required (handle conditional requirements)
      const isRequired = typeof rule.required === 'function' ? rule.required(config) : rule.required;
      
      if (isRequired) {
        // Run custom validator if provided
        if (rule.customValidator) {
          const result = rule.customValidator(config);
          if (!result.valid) {
            customErrors.push(result.message || `${rule.label || rule.field} is invalid`);
          }
        }
        
        // Check if required field is missing
        if (!config[rule.field]) {
          missingFields.push(rule.field);
          missingFieldLabels.push(rule.label || rule.field);
        }
      }
    });

    // Combine all errors
    const allErrors = [...missingFieldLabels, ...customErrors];
    
    if (allErrors.length > 0) {
      // Build specific error message
      const processorName = connector?.name;
      const errorMessage = missingConfigMessage || 
        ( processorName ? t('page.datasource.missing_config_tip', { name: processorName }) : `OAuth configuration issues: ${allErrors.join(', ')}`)

      window?.$modal?.confirm({
        title: t("common.tip"),
        icon: <ExclamationCircleOutlined />,
        content: errorMessage,
        okText: t("common.confirm"),
        cancelText: t("common.cancel"),
        onOk() {
          if (connector?.id) {
            nav(`/connector/edit/${connector.id}`, { state: connector });
          }
        },
      });
    } else {
      // Use custom connect URL or default to connector-specific endpoint
      const endpoint = connectUrl || `${window.location.origin}${window.location.pathname}connector/${connector?.id}/oauth_connect`;
      window.location.href = endpoint;
    }
  };

  return (
    <div className={className}>
      <Button type="primary" onClick={onConnectClick}>
        {connectButtonText || t("page.datasource.new.labels.connect")}
      </Button>
    </div>
  );
}