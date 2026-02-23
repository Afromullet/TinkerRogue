__________________________________________
#
# # # **IMPORTANT** # # # 


# Raid Package



- Either define archetypes.go in a json file, or use the encounter system. Consider creating a variation of the encounter system which weighs certain units more
    * I.E, "Ambush" - fast units have a weight, ranged battery prioritizes ranged units, etc. 



---------

- Add a "new type of attack" to combat, which is a heal. Use targeting cells to make things easier  

- renderables not created for new squad

- Untis not removed from roster after adding them to a squad

- Explore the go fix command



- Need better reward system. Both raids and regular combat should have rewards. We want rewards to be mostly the same, with additional weighing applies so that raids give better rewards


----
- Review all consts to see if they should be defined in a json file

- Start Random Encounter alwys starts the same encounter


- Either encounters or behavior seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Investigate whether it's the squad makup, or the weighting of attacks

- There has to be a tradeoff between casting spells and moving squads/engaging in combat. Doing both will feel too mechanical. 

- type EventType int in Overworld types is growing too large


 - System for gaining resources. Whether it's through battle or territory tbd


- Overworld node spawning is far too clustered together. All nodes spawn close to eachother. 


- Testing package has some functions which create initial player commanders and squads. move them to another package - probably bootstrap

- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Consider using an interface for intent in overworld


-  ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  

- Re-neable victory conditions in tickmanager.go. They are disabled for testing

- encounter service should not have mode coordinator. It overlaps too much with what the GUI is doing


- calculateEntryIndexAtPosition in squadeditor_movesquad.go is not accurately calculating the position to move the squad to. Check if there is an ebitenui var (such as entryselected) you can use instead of calculating with pixel positions


- Handle Entity Cleanup

- JSON file error handling


- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer

- Can always use the move undo command, even after attacking. That's a bug




________________________________________________



# Cleanup



- combatqueries.go still has functions which search the entire ecs space





# Review

## The following packages need review

- Guioverworld

- GUISquads
- GUIOverworld

- Worldmap
- Guiraids
- Raids
- combatpipeline
- Save System

____

x
- Look at all of the const. See which ones should be defined in a file





# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead
