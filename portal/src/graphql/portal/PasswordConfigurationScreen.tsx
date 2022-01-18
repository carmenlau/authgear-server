import React, { useCallback, useContext } from "react";
import { Context, FormattedMessage } from "@oursky/react-messageformat";
import { TextField, Toggle } from "@fluentui/react";
import cn from "classnames";
import produce from "immer";
import { clearEmptyObject } from "../../util/misc";
import { parseIntegerAllowLeadingZeros } from "../../util/input";
import { PortalAPIAppConfig } from "../../types";
import {
  AppConfigFormModel,
  useAppConfigForm,
} from "../../hook/useAppConfigForm";
import { useParams } from "react-router-dom";
import ShowLoading from "../../ShowLoading";
import ShowError from "../../ShowError";
import ScreenContent from "../../ScreenContent";
import ScreenTitle from "../../ScreenTitle";
import ScreenDescription from "../../ScreenDescription";
import WidgetTitle from "../../WidgetTitle";
import Widget from "../../Widget";
import FormContainer from "../../FormContainer";
import styles from "./PasswordConfigurationScreen.module.scss";

interface FormState {
  primaryAuthenticatorEnabled: boolean;
  codeExpirySeconds: number | undefined;
}

function constructFormState(config: PortalAPIAppConfig): FormState {
  return {
    primaryAuthenticatorEnabled:
      config.authentication?.primary_authenticators?.includes("password") ??
      false,
    codeExpirySeconds: config.forgot_password?.reset_code_expiry_seconds,
  };
}

function constructConfig(
  config: PortalAPIAppConfig,
  initialState: FormState,
  currentState: FormState
): PortalAPIAppConfig {
  return produce(config, (config) => {
    if (
      initialState.primaryAuthenticatorEnabled !==
      currentState.primaryAuthenticatorEnabled
    ) {
      config.authentication ??= {};
      config.authentication.primary_authenticators ??= [];

      if (config.authentication.primary_authenticators.includes("password")) {
        if (!currentState.primaryAuthenticatorEnabled) {
          config.authentication.primary_authenticators =
            config.authentication.primary_authenticators.filter(
              (p) => p !== "password"
            );
        }
      } else {
        if (currentState.primaryAuthenticatorEnabled) {
          config.authentication.primary_authenticators.push("password");
        }
      }
    }

    if (initialState.codeExpirySeconds !== currentState.codeExpirySeconds) {
      config.forgot_password = config.forgot_password ?? {};
      config.forgot_password.reset_code_expiry_seconds =
        currentState.codeExpirySeconds;
    }
    clearEmptyObject(config);
  });
}

interface PasswordConfigurationScreenContentProps {
  form: AppConfigFormModel<FormState>;
}

const PasswordConfigurationScreenContent: React.FC<PasswordConfigurationScreenContentProps> =
  function PasswordConfigurationScreenContent(props) {
    const { state, setState } = props.form;

    const { renderToString } = useContext(Context);

    const onPrimaryAuthenticatorEnabledChange = useCallback(
      (_, checked?: boolean) => {
        if (checked == null) {
          return;
        }
        setState((state) => ({
          ...state,
          primaryAuthenticatorEnabled: checked,
        }));
      },
      [setState]
    );

    const onCodeExpirySecondsChange = useCallback(
      (_, value?: string) => {
        setState((state) => ({
          ...state,
          codeExpirySeconds: parseIntegerAllowLeadingZeros(value),
        }));
      },
      [setState]
    );

    return (
      <ScreenContent>
        <ScreenTitle className={styles.widget}>
          <FormattedMessage id="PasswordConfigurationScreen.title" />
        </ScreenTitle>
        <ScreenDescription className={styles.widget}>
          <FormattedMessage id="PasswordConfigurationScreen.description" />
        </ScreenDescription>
        <Widget className={styles.widget}>
          <Toggle
            checked={state.primaryAuthenticatorEnabled}
            inlineLabel={true}
            label={
              <FormattedMessage id="PasswordConfigurationScreen.primary-authenticator-enabled.label" />
            }
            onChange={onPrimaryAuthenticatorEnabledChange}
          />
        </Widget>
        <Widget
          className={cn(styles.widget, {
            [styles.readOnly]: !state.primaryAuthenticatorEnabled,
          })}
        >
          <WidgetTitle>
            <FormattedMessage id="PasswordConfigurationScreen.code-settings" />
          </WidgetTitle>
          <TextField
            type="text"
            label={renderToString(
              "PasswordConfigurationScreen.reset-code-valid-duration.label"
            )}
            value={state.codeExpirySeconds?.toFixed(0) ?? ""}
            onChange={onCodeExpirySecondsChange}
          />
        </Widget>
        <ScreenDescription
          className={cn(styles.widget, {
            [styles.readOnly]: !state.primaryAuthenticatorEnabled,
          })}
        >
          <FormattedMessage
            id="PasswordConfigurationScreen.password-policy.description"
            values={{
              passwordPolicyPath: "../../password-policy",
            }}
          />
        </ScreenDescription>
      </ScreenContent>
    );
  };

const PasswordConfigurationScreenScreen: React.FC =
  function PasswordConfigurationScreenScreen() {
    const { appID } = useParams();
    const form = useAppConfigForm(appID, constructFormState, constructConfig);

    if (form.isLoading) {
      return <ShowLoading />;
    }

    if (form.loadError) {
      return <ShowError error={form.loadError} onRetry={form.reload} />;
    }

    return (
      <FormContainer form={form}>
        <PasswordConfigurationScreenContent form={form} />
      </FormContainer>
    );
  };

export default PasswordConfigurationScreenScreen;
