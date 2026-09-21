package com.oncode.server;

import com.oncode.server.agentruntime.AgentRuntimeModule;
import com.oncode.server.api.ApiModule;
import com.oncode.server.approval.ApprovalModule;
import com.oncode.server.context.ContextModule;
import com.oncode.server.inferencegateway.InferenceGatewayModule;
import com.oncode.server.persistence.PersistenceModule;
import com.oncode.server.toolgateway.ToolGatewayModule;
import com.oncode.server.workflow.WorkflowModule;

/** PHASE_02 modular-monolith package map. Not a runtime orchestrator. */
public final class ServerModules {

  public static final Class<?>[] CONFIGURATIONS = {
    ApiModule.class,
    WorkflowModule.class,
    ContextModule.class,
    ToolGatewayModule.class,
    AgentRuntimeModule.class,
    InferenceGatewayModule.class,
    ApprovalModule.class,
    PersistenceModule.class
  };

  private ServerModules() {}
}
