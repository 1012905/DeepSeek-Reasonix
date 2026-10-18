// @vitest-environment jsdom
import {act,cleanup,render,screen,waitFor} from "@testing-library/react";
import "./testkit";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {MockPort} from "../port/mock";
import {AgentCompositionPicker} from "./AgentCompositionPicker";
import {CompositionNewContext} from "./CompositionNewContext";
import {RunLine} from "./RunLine";
import { setShowsCompositionPicker } from "../state/compositionPicker";

beforeEach(() => setShowsCompositionPicker(true));
afterEach(cleanup);

it("hides the picker and performs no composition reads until explicitly enabled", async () => {
  setShowsCompositionPicker(false);
  const port = new MockPort();
  const read = vi.spyOn(port, "agentPresets");
  const select = vi.spyOn(port, "selectAgentComposition");
  render(<AgentCompositionPicker port={port} running={false} onChanged={()=>{}} onSettings={()=>{}} onError={()=>{}}/>);
  expect(screen.queryByRole("button")).toBeNull();
  expect(read).not.toHaveBeenCalled();
  act(() => setShowsCompositionPicker(true));
  await screen.findByRole("button", { name: /^Agent 组合/ });
  act(() => setShowsCompositionPicker(false));
  expect(screen.queryByRole("button")).toBeNull();
  expect(select).not.toHaveBeenCalled();
});

it("shows an unavailable state with a management entry when the catalog fails", async () => {
  const port = new MockPort();
  vi.spyOn(port, "agentPresets").mockRejectedValue(new Error("catalog unavailable"));
  const settings = vi.fn();
  render(<AgentCompositionPicker port={port} running={false} onChanged={()=>{}} onSettings={settings} onError={()=>{}}/>);
  const failed = await screen.findByRole("button", { name: "Agent 预设暂不可用" });
  await userEvent.click(failed);
  expect(settings).toHaveBeenCalledOnce();
});

it("groups composition controls with the run state without announcing controls as live status",async()=>{
  const port=new MockPort();const changed=vi.fn();const settings=vi.fn();
  const user=userEvent.setup();
  render(<RunLine label="空闲" running={false} blocked={false} sent={0} received={0} estimated={false}>
    <AgentCompositionPicker port={port} running={false} onChanged={changed} onSettings={settings} onError={()=>{}}/>
  </RunLine>);
  const picker=await screen.findByRole("button",{name:/^Agent 组合/});
  const status=screen.getByRole("status");
  expect(status.parentElement?.contains(picker)).toBe(true);
  expect(status.contains(picker)).toBe(false);
  expect(picker.compareDocumentPosition(status) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  expect(picker.textContent).toBe("标准模式");
  expect(screen.queryByRole("button",{name:"管理 Agent 预设"})).toBeNull();
  await user.click(picker);
  const review=screen.getByRole("menuitem",{name:/精简模式/});
  expect(review.getAttribute("data-action")).toBe("agent-composition.select");
  await user.click(review);
  expect(changed).toHaveBeenCalledOnce();
  expect(picker.getAttribute("aria-label")).toContain("精简模式");
  await user.click(picker);
  const manage=screen.getByRole("menuitem",{name:"管理 Agent 预设…"});
  expect(manage.getAttribute("data-action")).toBe("agent-composition.manage");
  await user.click(manage);
  expect(settings).toHaveBeenCalledOnce();
});

it("keeps started conversations locked and opens the chosen composition in a new conversation",async()=>{
  const port=new MockPort();const create=vi.fn();const select=vi.spyOn(port,"selectAgentComposition");
  const user=userEvent.setup();
  render(<CompositionNewContext.Provider value={create}><AgentCompositionPicker port={port} locked running={false} onChanged={()=>{}} onSettings={()=>{}} onError={()=>{}}/></CompositionNewContext.Provider>);
  const picker=await screen.findByRole("button",{name:/^Agent 组合/});
  expect(picker.textContent).toBe("标准模式");
  expect(picker.getAttribute("aria-label")).toContain("已锁定");
  await user.click(picker);
  expect(screen.getByText("该会话已开始，模式已锁定。请用其他模式新建会话。")).toBeTruthy();
  const review=screen.getByRole("menuitem",{name:/精简模式/});
  expect(review.getAttribute("data-action")).toBe("agent-composition.new");
  await user.click(review);
  expect(create).toHaveBeenCalledWith("builtin:minimal");
  expect(select).not.toHaveBeenCalled();
});

it("supports keyboard selection and Escape with focus returning to the one trigger",async()=>{
  const port=new MockPort();const select=vi.spyOn(port,"selectAgentComposition");
  const user=userEvent.setup();
  render(<AgentCompositionPicker port={port} running={false} onChanged={()=>{}} onSettings={()=>{}} onError={()=>{}}/>);
  const picker=await screen.findByRole("button",{name:/^Agent 组合/});
  picker.focus();await user.keyboard("{Enter}{ArrowDown}{Enter}");
  await waitFor(()=>expect(select).toHaveBeenCalledWith("builtin:minimal"));
  expect(document.activeElement).toBe(picker);
  await user.click(picker);await user.keyboard("{Escape}");
  expect(screen.queryByRole("menu")).toBeNull();
  expect(document.activeElement).toBe(picker);
});

it("keeps a failed selection unchanged and prevents duplicate submissions while pending",async()=>{
  const port=new MockPort();const error=vi.fn();const changed=vi.fn();
  let reject!:(error:Error)=>void;
  const select=vi.spyOn(port,"selectAgentComposition").mockImplementation(()=>new Promise((_,fail)=>{reject=fail;}));
  const user=userEvent.setup();
  render(<AgentCompositionPicker port={port} running={false} onChanged={changed} onSettings={()=>{}} onError={error}/>);
  const picker=await screen.findByRole("button",{name:/^Agent 组合/});
  await user.click(picker);await user.click(screen.getByRole("menuitem",{name:/精简模式/}));
  await user.click(picker);
  expect((screen.getByRole("menuitem",{name:/标准模式/}) as HTMLButtonElement).disabled).toBe(true);
  expect(document.activeElement).toBe(screen.getByRole("menuitem",{name:"管理 Agent 预设…"}));
  await user.click(screen.getByRole("menuitem",{name:/标准模式/}));
  expect(select).toHaveBeenCalledOnce();
  reject(new Error("fixture failure"));
  await waitFor(()=>expect(error).toHaveBeenCalledOnce());
  expect(picker.textContent).toContain("标准模式");
  expect(changed).not.toHaveBeenCalled();
});

it("keeps management accessible when a locked host cannot create conversations",async()=>{
  const port=new MockPort();const settings=vi.fn();const select=vi.spyOn(port,"selectAgentComposition");
  const user=userEvent.setup();
  render(<AgentCompositionPicker port={port} locked running={false} onChanged={()=>{}} onSettings={settings} onError={()=>{}}/>);
  await user.click(await screen.findByRole("button",{name:/^Agent 组合.*已锁定/}));
  expect(screen.getAllByRole("menuitem")).toHaveLength(1);
  await user.click(screen.getByRole("menuitem",{name:"管理 Agent 预设…"}));
  expect(settings).toHaveBeenCalledOnce();
  expect(select).not.toHaveBeenCalled();
});

it("distinguishes equal names by source and retains a deleted binding's name",async()=>{
  const port=new MockPort();const catalog=await port.agentPresets();
  const original=catalog.presets[0];
  vi.spyOn(port,"agentPresets").mockResolvedValue({...catalog,presets:[
    {...original,definition:{...original.definition,id:"user:review",name:"同名审查"}},
    {...original,definition:{...original.definition,id:"project:review",name:"同名审查"}},
  ]});
  vi.spyOn(port,"agentComposition").mockResolvedValue({id:"user:deleted",name:"已删除但绑定的预设",digest:"frozen",revision:"1",started:false});
  const user=userEvent.setup();
  render(<AgentCompositionPicker port={port} running={false} onChanged={()=>{}} onSettings={()=>{}} onError={()=>{}}/>);
  const trigger=await screen.findByRole("button",{name:/^Agent 组合.*已删除但绑定的预设/});
  await user.click(trigger);
  const rows=screen.getAllByRole("menuitem",{name:/同名审查/});
  expect(rows.map(row=>row.textContent)).toEqual([expect.stringContaining("个人"),expect.stringContaining("项目")]);
});
