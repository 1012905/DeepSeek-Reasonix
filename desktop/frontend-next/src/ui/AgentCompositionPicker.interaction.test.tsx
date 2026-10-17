// @vitest-environment jsdom
import {cleanup,render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,expect,it,vi} from "vitest";
import {MockPort} from "../port/mock";
import {AgentCompositionPicker} from "./AgentCompositionPicker";
import {CompositionNewContext} from "./CompositionNewContext";

afterEach(cleanup);

it("keeps started conversations locked and opens the chosen composition in a new conversation",async()=>{
  const port=new MockPort();const create=vi.fn();const select=vi.spyOn(port,"selectAgentComposition");
  const user=userEvent.setup();
  render(<CompositionNewContext.Provider value={create}><AgentCompositionPicker port={port} locked running={false} onChanged={()=>{}} onSettings={()=>{}} onError={()=>{}}/></CompositionNewContext.Provider>);
  const picker=await screen.findByRole<HTMLSelectElement>("combobox",{name:"Agent 组合"});
  expect(picker.disabled).toBe(true);
  await user.selectOptions(screen.getByRole("combobox",{name:"用其他组合新建会话"}),"builtin:review");
  await user.click(screen.getByRole("button",{name:"新建会话"}));
  expect(create).toHaveBeenCalledWith("builtin:review");
  expect(select).not.toHaveBeenCalled();
});
