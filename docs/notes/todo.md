# Perks and Artifacts

- Determine whether there is a better way to integrate perks and artifacts into combat. Want to make it cleaner, rather than have the checks littered throughout the code. 
- Then do the same for spells

# Documentation cleanup after perks are wrapped up

- Need to update ARTIFACT_SYSTEM.md with how to add new artifacts
- Need to update PERK_SYSTEM.md with how to add new perks
- Need to update SPELL_DOCUMENTATION on how to add new spells
- ARCHITECTURE_LAYERS.md
- GAMEDATA_OVERVIEW
- HOOKS_AND_CALLBACKS.md
- ENTITY_REFERENCE.md
- DATA_FLOW_PATTERNS.md
- ARTIFACT_SYSTEM.md
- PERK_SYSTEM.md
- DOCUMENTATION.md

# Perk System Cleanup

-ExecuteAttackAction is growing too large
- calculateDamage is growing too large
- Need better perk creation and registration system
- Need a GUI for perks
- Need a perk unlock system. Should be designed side by side with spell unlock system

# Future AI cleanup

- Need to take artifacts and perks into account for the utiltiy map calculations

# Spells

- Leader can select from 1 of n spells on level up. Also can select from perks. Either or choice
- Consider a different mana regeneration system. Currently, mana is regenerated after combat. Maybe regenerate it turn by turn, or regenerate it from attacking

# Bug Fixes

- XP Awards seem to be odd. I.E, didivided by 10. Combat resolution grants 40 xp, but units only get 4 xp
- Can cast spells and use artifacts on enemy turn




# GUI Updates

- Squad Edit Mode requires me to select a unit from the panel on the right before Removing, Making it a leader, or viewing the unit. I also want to be able to access that by selecting the unit in the grid
- Unit Purchase Mode needs to show Unit Info

# JSON

- Add Error Checking for JSON. I.E, a "Support" unit has to have a "Heal" attack type, and so on. Determine valid state
- Review other JSON files and determine valid state and do load-time checking.


# Input

- Allow controsl to be remapped

# Combat


- Add Zones of Control


# Action Evaluator

# Other 

- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer
- ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  
- Add Debug command to reload JSON. Will be used for testing
- I added a json file for raid room archetypes - gamedata\raidarchetypes.go. I will have to add some more randomization to the archetypes. One possibility is through unit traits. Not sure exactly yet how to implement that..Might just go with a "preferred room" field for the unit json
- Need to be able to go back to the main menu from overworld or roguelike mode. 

# Cleanup

- combatprocessing.go has grown too fragemented. We have several "processatttacks" functions. We need to make it cleaner


- Determine whether you can use DirtyCache as an interface for all of the other caches. Also determine whether we really need the caches

- Make sure entities are cleaned up upon destruction. Need to determine what entities have a "lifecycle" by determining what addcomponent is called on. 

- combatqueries.go still has functions which search the entire ecs space


- JSON file error handling

- Testing package has some functions which create initial player commanders and squads. move them to another package - probably bootstrap

- type EventType int in Overworld types is growing too large

- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Re-enable victory conditions in tickmanager.go. They are disabled for testing

- Destroyed factions need to be removed from combat


# Review

- Either encounters or behavior seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Investigate whether it's the squad makup, or the weighting of attacks. Find a way to measure how effective the encounter creation and threat maps are
- Determine if GUI state management is clean enough

# Review

- Review GUi

# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead. Check into logging to see if we can disable them if possible. 


# Research 

- Explore the go fix command


# Overworld (Defer)

- System for gaining overworld resources. Whether it's through battle or territory tbd
- Overworld node spawning is far too clustered together. All nodes spawn close to eachother. 
- Consider using an interface for intent in overworld